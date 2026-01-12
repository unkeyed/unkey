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

// Decrypt decrypts ciphertext using AES-GCM with the provided key and nonce.
//
// Returns an error if validation fails or if the ciphertext has been tampered with.
func Decrypt(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {

	if err := assert.All(
		assert.Equal(len(nonce), 12, "nonce must be 12 bytes for AES-GCM"),
		assert.Equal(len(key), 32, "key size must be 32 bytes"),
	); err != nil {
		return nil, err
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
