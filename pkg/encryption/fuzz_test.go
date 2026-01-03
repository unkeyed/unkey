package encryption_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/encryption"
)

// FuzzEncryptDecrypt tests the round-trip encryption and decryption
func FuzzEncryptDecrypt(f *testing.F) {
	// Add some seed corpus
	f.Add([]byte("16-byte test key!"), []byte("hello world"))
	f.Add([]byte("24-byte key for testing!!"), []byte("Lorem ipsum dolor sit amet"))
	f.Add([]byte("32-byte key for thorough testing!!!"), []byte(""))
	f.Add([]byte("16-byte test key!"), []byte{0, 1, 2, 3, 4, 5})

	f.Fuzz(func(t *testing.T, key, plaintext []byte) {
		// Skip invalid key sizes - AES requires keys of 16, 24, or 32 bytes
		if len(key) != 16 && len(key) != 24 && len(key) != 32 {
			t.Skip("Skipping invalid key size")
		}

		// Encrypt the plaintext
		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err, "Encryption failed")

		// Verify nonce length
		require.Equal(t, 12, len(nonce), "Nonce should be 12 bytes")

		// Decrypt the ciphertext
		decrypted, err := encryption.Decrypt(key, nonce, ciphertext)
		require.NoError(t, err, "Decryption failed")

		// Verify the decrypted data matches the original plaintext
		require.True(t, bytes.Equal(plaintext, decrypted), "Decrypted text doesn't match original plaintext")
	})
}

// FuzzDecryptWithWrongKey tests that decryption with a different key fails
func FuzzDecryptWithWrongKey(f *testing.F) {
	f.Add(
		[]byte("16-byte test key!"),
		[]byte("different key..."),
		[]byte("test plaintext"),
	)

	f.Fuzz(func(t *testing.T, encryptKey, decryptKey, plaintext []byte) {
		// Skip if keys are the same or invalid sizes
		if bytes.Equal(encryptKey, decryptKey) {
			t.Skip("Skipping identical keys")
		}
		if (len(encryptKey) != 16 && len(encryptKey) != 24 && len(encryptKey) != 32) ||
			(len(decryptKey) != 16 && len(decryptKey) != 24 && len(decryptKey) != 32) {
			t.Skip("Skipping invalid key sizes")
		}

		// Encrypt with first key
		nonce, ciphertext, err := encryption.Encrypt(encryptKey, plaintext)
		if err != nil {
			t.Skip("Encryption failed")
		}

		// Decrypt with different key - should fail
		_, err = encryption.Decrypt(decryptKey, nonce, ciphertext)
		require.Error(t, err, "Decryption should fail with wrong key")
	})
}

// FuzzTamperedCiphertext tests that modified ciphertext fails to decrypt correctly
func FuzzTamperedCiphertext(f *testing.F) {
	f.Add([]byte("16-byte test key!"), []byte("test plaintext"), byte(1), uint16(0))

	f.Fuzz(func(t *testing.T, key, plaintext []byte, tamperedByte byte, position uint16) {
		// Skip invalid key sizes
		if len(key) != 16 && len(key) != 24 && len(key) != 32 {
			t.Skip("Skipping invalid key size")
		}

		// Skip empty plaintext
		if len(plaintext) == 0 {
			t.Skip("Skipping empty plaintext")
		}

		// Encrypt the plaintext
		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		if err != nil {
			t.Skip("Encryption failed")
		}

		// Need at least one byte of ciphertext to tamper with
		if len(ciphertext) == 0 {
			t.Skip("Ciphertext too short")
		}

		// Create a copy of the ciphertext and tamper with it
		tamperedCiphertext := make([]byte, len(ciphertext))
		copy(tamperedCiphertext, ciphertext)

		// Modify one byte
		pos := int(position) % len(tamperedCiphertext)
		tamperedCiphertext[pos] ^= tamperedByte

		// Skip if our tampering didn't actually change the byte
		if tamperedCiphertext[pos] == ciphertext[pos] {
			t.Skip("Tampering didn't change the ciphertext")
		}

		// Attempt to decrypt the tampered ciphertext
		_, err = encryption.Decrypt(key, nonce, tamperedCiphertext)
		require.Error(t, err, "Decryption should fail with tampered ciphertext")
	})
}

// FuzzTamperedNonce tests that a modified nonce fails to decrypt correctly
func FuzzTamperedNonce(f *testing.F) {
	f.Add([]byte("16-byte test key!"), []byte("test plaintext"), byte(1), uint16(0))

	f.Fuzz(func(t *testing.T, key, plaintext []byte, tamperedByte byte, position uint16) {
		// Skip invalid key sizes
		if len(key) != 16 && len(key) != 24 && len(key) != 32 {
			t.Skip("Skipping invalid key size")
		}

		// Encrypt the plaintext
		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		if err != nil {
			t.Skip("Encryption failed")
		}

		// Create a copy of the nonce and tamper with it
		tamperedNonce := make([]byte, len(nonce))
		copy(tamperedNonce, nonce)

		// Modify one byte
		pos := int(position) % len(tamperedNonce)
		tamperedNonce[pos] ^= tamperedByte

		// Skip if our tampering didn't actually change the byte
		if tamperedNonce[pos] == nonce[pos] {
			t.Skip("Tampering didn't change the nonce")
		}

		// Attempt to decrypt with the tampered nonce
		_, err = encryption.Decrypt(key, tamperedNonce, ciphertext)
		require.Error(t, err, "Decryption should fail with tampered nonce")
	})
}
