package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"strings"
)

// SealCredentials stores secrets with authenticated encryption. The key stays
// outside the database and is supplied through CRM_ENCRYPTION_KEY.
func SealCredentials(plaintext, keyHex string) (string, error) {
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return "", errors.New("CRM_ENCRYPTION_KEY must contain 32 bytes encoded as hex")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), []byte("crm-integration-v1"))
	return "enc:v1:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// UnsealCredentials reverses SealCredentials. It returns an error for malformed
// input or when authentication fails (wrong key or tampered ciphertext).
func UnsealCredentials(sealed, keyHex string) (string, error) {
	if !strings.HasPrefix(sealed, "enc:v1:") {
		return "", errors.New("unsupported credential format")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(sealed, "enc:v1:"))
	if err != nil {
		return "", errors.New("invalid credential encoding")
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil || len(key) != 32 {
		return "", errors.New("CRM_ENCRYPTION_KEY must contain 32 bytes encoded as hex")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	plain, err := aead.Open(nil, nonce, ciphertext, []byte("crm-integration-v1"))
	if err != nil {
		return "", errors.New("credential decryption failed")
	}
	return string(plain), nil
}
