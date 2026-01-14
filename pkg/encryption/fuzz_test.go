package encryption_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzEncryptDecrypt tests the round-trip encryption and decryption.
func FuzzEncryptDecrypt(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		key := c.BytesN(32)
		plaintext := c.Bytes()

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

// FuzzDecryptWithWrongKey tests that decryption with a different key fails.
func FuzzDecryptWithWrongKey(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		// Generate two different valid AES keysi
		key1 := c.BytesN(32)
		key2 := c.BytesN(32)

		// Skip if keys are the same
		if bytes.Equal(key1, key2) {
			t.Skip("Keys are identical")
		}

		plaintext := c.Bytes()

		// Encrypt with first key
		nonce, ciphertext, err := encryption.Encrypt(key1, plaintext)
		require.NoError(t, err, "Encryption failed")

		// Decrypt with different key - should fail
		_, err = encryption.Decrypt(key2, nonce, ciphertext)
		require.Error(t, err, "Decryption should fail with wrong key")
	})
}

// FuzzTamperedCiphertext tests that modified ciphertext fails to decrypt correctly.
func FuzzTamperedCiphertext(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		key := c.BytesN(32)

		plaintext := c.Bytes()
		if len(plaintext) == 0 {
			t.Skip("Skipping empty plaintext")
		}

		tamperedByte := c.Uint8()
		if tamperedByte == 0 {
			t.Skip("XOR with 0 doesn't change anything")
		}
		position := c.Uint16()

		// Encrypt the plaintext
		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err, "Encryption failed")

		if len(ciphertext) == 0 {
			t.Skip("Ciphertext too short")
		}

		// Create a copy of the ciphertext and tamper with it
		tamperedCiphertext := make([]byte, len(ciphertext))
		copy(tamperedCiphertext, ciphertext)

		// Modify one byte
		pos := int(position) % len(tamperedCiphertext)
		tamperedCiphertext[pos] ^= tamperedByte

		// Attempt to decrypt the tampered ciphertext
		_, err = encryption.Decrypt(key, nonce, tamperedCiphertext)
		require.Error(t, err, "Decryption should fail with tampered ciphertext")
	})
}

// FuzzTamperedNonce tests that a modified nonce fails to decrypt correctly.
func FuzzTamperedNonce(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)

		key := c.BytesN(32)
		plaintext := c.Bytes()

		tamperedByte := c.Uint8()
		if tamperedByte == 0 {
			t.Skip("XOR with 0 doesn't change anything")
		}
		position := c.Uint16()

		// Encrypt the plaintext
		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err, "Encryption failed")

		// Create a copy of the nonce and tamper with it
		tamperedNonce := make([]byte, len(nonce))
		copy(tamperedNonce, nonce)

		// Modify one byte
		pos := int(position) % len(tamperedNonce)
		tamperedNonce[pos] ^= tamperedByte

		// Attempt to decrypt with the tampered nonce
		_, err = encryption.Decrypt(key, tamperedNonce, ciphertext)
		require.Error(t, err, "Decryption should fail with tampered nonce")
	})
}
