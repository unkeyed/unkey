package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/encryption"
)

func TestEncryptDecrypt(t *testing.T) {

	data := make([]byte, 1024)
	_, err := rand.Read(data)
	require.NoError(t, err)

	key := make([]byte, 32)
	_, err = rand.Read(key)
	require.NoError(t, err)

	nonce, ciphertext, err := encryption.Encrypt(key, data)
	require.NoError(t, err)

	plaintext, err := encryption.Decrypt(key, nonce, ciphertext)
	require.NoError(t, err)
	require.Equal(t, data, plaintext)

}
