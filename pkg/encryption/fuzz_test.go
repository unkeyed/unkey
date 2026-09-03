package encryption_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/encryption"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

// FuzzEncryptDecrypt verifies that encryption preserves arbitrary plaintext.
func FuzzEncryptDecrypt(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		key := c.BytesN(32)
		plaintext := c.Bytes()

		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err)
		require.Len(t, nonce, 12)

		decrypted, err := encryption.Decrypt(key, nonce, ciphertext)
		require.NoError(t, err)
		require.True(t, bytes.Equal(plaintext, decrypted))
	})
}

// FuzzDecryptWithWrongKey verifies that Decrypt rejects a key with one changed bit.
func FuzzDecryptWithWrongKey(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		key := c.BytesN(32)
		plaintext := c.Bytes()
		position := int(c.Uint8()) % len(key)
		bit := byte(1 << (c.Uint8() % 8))

		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err)

		wrongKey := bytes.Clone(key)
		wrongKey[position] ^= bit

		decrypted, err := encryption.Decrypt(wrongKey, nonce, ciphertext)
		require.Error(t, err)
		require.Nil(t, decrypted)
	})
}

// FuzzTamperedCiphertext verifies that Decrypt rejects ciphertext with one changed bit.
func FuzzTamperedCiphertext(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		key := c.BytesN(32)
		plaintext := c.Bytes()
		position := c.Uint16()
		bit := byte(1 << (c.Uint8() % 8))

		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err)

		tamperedCiphertext := bytes.Clone(ciphertext)
		tamperedCiphertext[int(position)%len(tamperedCiphertext)] ^= bit

		decrypted, err := encryption.Decrypt(key, nonce, tamperedCiphertext)
		require.Error(t, err)
		require.Nil(t, decrypted)
	})
}

// FuzzTamperedNonce verifies that Decrypt rejects a nonce with one changed bit.
func FuzzTamperedNonce(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		key := c.BytesN(32)
		plaintext := c.Bytes()
		position := c.Uint8()
		bit := byte(1 << (c.Uint8() % 8))

		nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
		require.NoError(t, err)

		tamperedNonce := bytes.Clone(nonce)
		tamperedNonce[int(position)%len(tamperedNonce)] ^= bit

		decrypted, err := encryption.Decrypt(key, tamperedNonce, ciphertext)
		require.Error(t, err)
		require.Nil(t, decrypted)
	})
}
