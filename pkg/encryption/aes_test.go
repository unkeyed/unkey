package encryption_test

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/encryption"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "empty plaintext",
			plaintext: []byte{},
		},
		{
			name:      "one MiB plaintext",
			plaintext: bytes.Repeat([]byte("secret"), (1<<20)/len("secret")+1)[:1<<20],
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := bytes.Repeat([]byte{0x42}, 32)

			nonce, ciphertext, err := encryption.Encrypt(key, test.plaintext)
			require.NoError(t, err)
			require.Len(t, nonce, 12)
			require.Len(t, ciphertext, len(test.plaintext)+16)

			plaintext, err := encryption.Decrypt(key, nonce, ciphertext)
			require.NoError(t, err)
			require.True(t, bytes.Equal(test.plaintext, plaintext))
		})
	}
}

func TestEncrypt_ProducesStandardAES256GCMCiphertext(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	want := []byte("customer secret")

	nonce, ciphertext, err := encryption.Encrypt(key, want)
	require.NoError(t, err)

	block, err := aes.NewCipher(key)
	require.NoError(t, err)
	gcm, err := cipher.NewGCM(block)
	require.NoError(t, err)

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	require.NoError(t, err)
	require.Equal(t, want, plaintext)
}

// TestEncrypt_UsesFreshNonces protects AES-GCM from catastrophic nonce reuse.
func TestEncrypt_UsesFreshNonces(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	plaintext := []byte("customer secret")

	firstNonce, firstCiphertext, err := encryption.Encrypt(key, plaintext)
	require.NoError(t, err)
	secondNonce, secondCiphertext, err := encryption.Encrypt(key, plaintext)
	require.NoError(t, err)

	require.Len(t, firstNonce, 12)
	require.Len(t, secondNonce, 12)
	require.NotEqual(t, firstNonce, secondNonce)
	require.NotEqual(t, firstCiphertext, secondCiphertext)
}

func TestEncryptDecrypt_AcceptsAES256Key(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	want := []byte("customer secret")

	nonce, ciphertext, err := encryption.Encrypt(key, want)
	require.NoError(t, err)

	plaintext, err := encryption.Decrypt(key, nonce, ciphertext)
	require.NoError(t, err)
	require.Equal(t, want, plaintext)
}

func TestEncryptDecrypt_RejectsNonAES256KeySizes(t *testing.T) {
	for _, keySize := range []int{0, 15, 16, 24, 31, 33} {
		t.Run(strconv.Itoa(keySize), func(t *testing.T) {
			key := make([]byte, keySize)

			var encryptErr error
			require.NotPanics(t, func() {
				_, _, encryptErr = encryption.Encrypt(key, []byte("customer secret"))
			})
			require.ErrorContains(t, encryptErr, "key size must be 32 bytes")

			var decryptErr error
			require.NotPanics(t, func() {
				_, decryptErr = encryption.Decrypt(key, make([]byte, 12), make([]byte, 16))
			})
			require.ErrorContains(t, decryptErr, "key size must be 32 bytes")
		})
	}
}

// TestDecrypt_KnownAES256GCMVector verifies compatibility with standard AES-GCM.
func TestDecrypt_KnownAES256GCMVector(t *testing.T) {
	// The vector comes from the Go crypto/cipher GCM test suite. Its expected
	// bytes are independent of this package.
	key := decodeHex(t, "feffe9928665731c6d6a8f9467308308feffe9928665731c6d6a8f9467308308")
	nonce := decodeHex(t, "54cc7dc2c37ec006bcc6d1da")
	ciphertext := decodeHex(t, "d50b9e252b70945d4240d351677eb10f937cdaef6f2822b6a3191654ba41b197")
	want := decodeHex(t, "007c5e5b3e59df24a7c355584fc1518d")

	plaintext, err := encryption.Decrypt(key, nonce, ciphertext)
	require.NoError(t, err)
	require.Equal(t, want, plaintext)
}

func TestDecrypt_RejectsWrongKey(t *testing.T) {
	key, nonce, ciphertext := encryptSecret(t, []byte("customer secret"))
	wrongKey := bytes.Clone(key)
	wrongKey[0] ^= 1

	plaintext, err := encryption.Decrypt(wrongKey, nonce, ciphertext)
	require.Error(t, err)
	require.Nil(t, plaintext)
}

func TestDecrypt_RejectsWrongNonce(t *testing.T) {
	key, nonce, ciphertext := encryptSecret(t, []byte("customer secret"))
	wrongNonce := bytes.Clone(nonce)
	wrongNonce[0] ^= 1

	plaintext, err := encryption.Decrypt(key, wrongNonce, ciphertext)
	require.Error(t, err)
	require.Nil(t, plaintext)
}

func TestDecrypt_RejectsInvalidNonceSizes(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)

	for _, nonceSize := range []int{0, 11, 13} {
		t.Run(strconv.Itoa(nonceSize), func(t *testing.T) {
			var plaintext []byte
			var err error
			require.NotPanics(t, func() {
				plaintext, err = encryption.Decrypt(key, make([]byte, nonceSize), make([]byte, 16))
			})
			require.ErrorContains(t, err, "nonce must be 12 bytes for AES-GCM")
			require.Nil(t, plaintext)
		})
	}
}

// TestDecrypt_RejectsCiphertextMutation protects ciphertext integrity and authenticity.
func TestDecrypt_RejectsCiphertextMutation(t *testing.T) {
	key, nonce, ciphertext := encryptSecret(t, []byte("customer secret"))
	tests := []struct {
		name  string
		index int
	}{
		{
			name:  "encrypted payload bit",
			index: 0,
		},
		{
			name:  "authentication tag bit",
			index: len(ciphertext) - 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutatedCiphertext := bytes.Clone(ciphertext)
			mutatedCiphertext[test.index] ^= 1

			plaintext, err := encryption.Decrypt(key, nonce, mutatedCiphertext)
			require.Error(t, err)
			require.Nil(t, plaintext)
		})
	}
}

func TestDecrypt_RejectsCiphertextShorterThanTag(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	nonce := make([]byte, 12)

	for _, test := range []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "empty",
			ciphertext: []byte{},
		},
		{
			name:       "fifteen bytes",
			ciphertext: make([]byte, 15),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			plaintext, err := encryption.Decrypt(key, nonce, test.ciphertext)
			require.Error(t, err)
			require.Nil(t, plaintext)
		})
	}
}

func encryptSecret(t *testing.T, plaintext []byte) (key []byte, nonce []byte, ciphertext []byte) {
	t.Helper()

	key = bytes.Repeat([]byte{0x42}, 32)
	nonce, ciphertext, err := encryption.Encrypt(key, plaintext)
	require.NoError(t, err)

	return key, nonce, ciphertext
}

func decodeHex(t *testing.T, value string) []byte {
	t.Helper()

	decoded, err := hex.DecodeString(value)
	require.NoError(t, err)
	return decoded
}
