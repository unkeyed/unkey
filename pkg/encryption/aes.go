package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"github.com/unkeyed/unkey/pkg/assert"
)

func Encrypt(key []byte, plaintext []byte) (nonce []byte, ciphertext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	nonce = make([]byte, 12)
	n, err := rand.Read(nonce)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create nonce: %w", err)
	}
	if n != 12 {
		return nil, nil, fmt.Errorf("failed to read 12 bytes of random data: %w", err)
	}

	aes, err := cipher.NewGCM(block)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to create gcm: %w", err)
	}
	ciphertext = aes.Seal(nil, nonce, plaintext, nil)

	return nonce, ciphertext, nil

}

// gcmNonceSize is the required nonce size for AES-GCM (96 bits).
const gcmNonceSize = 12

// validAESKeySize checks if the key length is valid for AES (16, 24, or 32 bytes).
func validAESKeySize(keyLen int) bool {
	return keyLen == 16 || keyLen == 24 || keyLen == 32
}

// Decrypt decrypts ciphertext using AES-GCM with the provided key and nonce.
//
// The key must be 16, 24, or 32 bytes (AES-128, AES-192, or AES-256).
// The nonce must be exactly 12 bytes (standard GCM nonce size).
// Returns an error if validation fails or if the ciphertext has been tampered with.
func Decrypt(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	// Defense in depth: validate inputs before passing to crypto library.
	// These assertions prevent panics from malformed input.
	if err := assert.Equal(len(nonce), gcmNonceSize, fmt.Sprintf("invalid nonce length: expected %d bytes, got %d", gcmNonceSize, len(nonce))); err != nil {
		return nil, err
	}
	if !validAESKeySize(len(key)) {
		return nil, fmt.Errorf("invalid key length: expected 16, 24, or 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcm: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}
