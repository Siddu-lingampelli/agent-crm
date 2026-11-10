package security

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"
)

func TestCredentialsRequireKeyAndEncryptWithAuthentication(t *testing.T) {
	if _, err := SealCredentials("secret", ""); err == nil {
		t.Fatal("allowed missing encryption key")
	}
	key := strings.Repeat("01", 32)
	sealed, err := SealCredentials("secret", key)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(sealed, "secret") {
		t.Fatal("plaintext persisted")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sealed, "enc:v1:"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, _ := hex.DecodeString(key)
	block, _ := aes.NewCipher(decoded)
	aead, _ := cipher.NewGCM(block)
	plaintext, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("crm-integration-v1"))
	if err != nil || string(plaintext) != "secret" {
		t.Fatalf("cannot decrypt stored credential: %v", err)
	}
	raw[len(raw)-1] ^= 1
	if _, err := aead.Open(nil, raw[:aead.NonceSize()], raw[aead.NonceSize():], []byte("crm-integration-v1")); err == nil {
		t.Fatal("tampered credential accepted")
	}
}
