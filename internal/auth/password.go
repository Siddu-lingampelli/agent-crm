// Password hashing using Argon2id. The encoded hash is in PHC string format
// so it can be verified after a restart without extra state.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time     = 1
	argon2Memory   = 64 * 1024 // 64 MiB
	argon2Threads  = 4
	argon2KeyLen   = 32
	argon2SaltLen  = 16
	prefix         = "$argon2id$"
	memoryOverhead = argon2Memory
)

// Bound concurrent 64 MiB derivations so parallel logins cannot exhaust RAM.
var passwordSlots = make(chan struct{}, 2)

// HashArgon2 produces a PHC-format Argon2id hash of password.
func HashArgon2(password string) (string, error) {
	passwordSlots <- struct{}{}
	defer func() { <-passwordSlots }()
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	encoded := fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash))

	return encoded, nil
}

// VerifyArgon2 checks password against an Argon2id PHC hash.
func VerifyArgon2(hashed, password string) (bool, error) {
	passwordSlots <- struct{}{}
	defer func() { <-passwordSlots }()
	if !strings.HasPrefix(hashed, prefix) {
		return false, errors.New("unsupported hash format")
	}

	parts := strings.Split(hashed, "$")
	if len(parts) != 6 {
		return false, errors.New("malformed hash")
	}
	if parts[2] != "v=19" {
		return false, errors.New("unsupported argon2 version")
	}

	var memory, time, threads uint32
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, err
	}
	if memory < 8 || memory > 256*1024 || time < 1 || time > 10 || threads < 1 || threads > 32 || memory < 8*threads {
		return false, errors.New("invalid argon2 parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("decode salt: %w", err)
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("decode hash: %w", err)
	}
	if len(salt) < 8 || len(salt) > 64 || len(want) != argon2KeyLen {
		return false, errors.New("invalid argon2 salt or hash length")
	}

	got := argon2.IDKey([]byte(password), salt, time, memory, uint8(threads), argon2KeyLen)

	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
