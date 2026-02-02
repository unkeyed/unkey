package encryption_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/encryption"
)

func TestWorkspaceEncryption_EncryptDecrypt(t *testing.T) {
	t.Run("encrypts and decrypts token successfully", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		workspaceID := "ws_test123"
		token := "sk_test_abcdef123456"

		encrypted, err := we.EncryptToken(workspaceID, token)
		require.NoError(t, err)
		require.NotEmpty(t, encrypted)
		require.NotEqual(t, token, encrypted)

		decrypted, err := we.DecryptToken(workspaceID, encrypted)
		require.NoError(t, err)
		require.Equal(t, token, decrypted)
	})

	t.Run("different workspaces produce different ciphertexts", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		token := "sk_test_same_token"
		workspace1 := "ws_workspace1"
		workspace2 := "ws_workspace2"

		encrypted1, err := we.EncryptToken(workspace1, token)
		require.NoError(t, err)

		encrypted2, err := we.EncryptToken(workspace2, token)
		require.NoError(t, err)

		// Same token encrypted for different workspaces should produce different ciphertexts
		require.NotEqual(t, encrypted1, encrypted2)

		// Each should decrypt correctly with its own workspace ID
		decrypted1, err := we.DecryptToken(workspace1, encrypted1)
		require.NoError(t, err)
		require.Equal(t, token, decrypted1)

		decrypted2, err := we.DecryptToken(workspace2, encrypted2)
		require.NoError(t, err)
		require.Equal(t, token, decrypted2)
	})

	t.Run("cannot decrypt with wrong workspace ID", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		token := "sk_test_token"
		workspace1 := "ws_workspace1"
		workspace2 := "ws_workspace2"

		encrypted, err := we.EncryptToken(workspace1, token)
		require.NoError(t, err)

		// Attempting to decrypt with wrong workspace ID should fail
		_, err = we.DecryptToken(workspace2, encrypted)
		require.Error(t, err)
	})

	t.Run("same token encrypted twice produces different ciphertexts", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		workspaceID := "ws_test"
		token := "sk_test_token"

		encrypted1, err := we.EncryptToken(workspaceID, token)
		require.NoError(t, err)

		encrypted2, err := we.EncryptToken(workspaceID, token)
		require.NoError(t, err)

		// Due to random nonce, same token encrypted twice should differ
		require.NotEqual(t, encrypted1, encrypted2)

		// Both should decrypt to the same token
		decrypted1, err := we.DecryptToken(workspaceID, encrypted1)
		require.NoError(t, err)
		require.Equal(t, token, decrypted1)

		decrypted2, err := we.DecryptToken(workspaceID, encrypted2)
		require.NoError(t, err)
		require.Equal(t, token, decrypted2)
	})

	t.Run("handles long tokens", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		workspaceID := "ws_test"
		// Create a long token (1KB)
		longToken := make([]byte, 1024)
		_, err = rand.Read(longToken)
		require.NoError(t, err)
		token := string(longToken)

		encrypted, err := we.EncryptToken(workspaceID, token)
		require.NoError(t, err)

		decrypted, err := we.DecryptToken(workspaceID, encrypted)
		require.NoError(t, err)
		require.Equal(t, token, decrypted)
	})
}

func TestWorkspaceEncryption_ValidationErrors(t *testing.T) {
	t.Run("rejects master key shorter than 32 bytes", func(t *testing.T) {
		shortKey := make([]byte, 16)
		_, err := rand.Read(shortKey)
		require.NoError(t, err)

		_, err = encryption.NewWorkspaceEncryption(shortKey)
		require.Error(t, err)
		require.Contains(t, err.Error(), "master key must be at least 32 bytes")
	})

	t.Run("rejects empty token for encryption", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		_, err = we.EncryptToken("ws_test", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "token cannot be empty")
	})

	t.Run("rejects empty workspace ID for encryption", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		_, err = we.EncryptToken("", "sk_test_token")
		require.Error(t, err)
		require.Contains(t, err.Error(), "workspace ID cannot be empty")
	})

	t.Run("rejects empty encrypted token for decryption", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		_, err = we.DecryptToken("ws_test", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "encrypted token cannot be empty")
	})

	t.Run("rejects malformed encrypted token", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		// Missing separator
		_, err = we.DecryptToken("ws_test", "invalidbase64data")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid encrypted token format")
	})

	t.Run("rejects invalid base64 in nonce", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		// Invalid base64 in nonce part
		_, err = we.DecryptToken("ws_test", "!!!invalid!!!:validbase64==")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode nonce")
	})

	t.Run("rejects invalid base64 in ciphertext", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		// Invalid base64 in ciphertext part
		_, err = we.DecryptToken("ws_test", "validbase64==:!!!invalid!!!")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to decode")
	})
}

func TestWorkspaceEncryption_KeyDerivation(t *testing.T) {
	t.Run("same workspace ID always derives same key", func(t *testing.T) {
		masterKey := make([]byte, 32)
		_, err := rand.Read(masterKey)
		require.NoError(t, err)

		we, err := encryption.NewWorkspaceEncryption(masterKey)
		require.NoError(t, err)

		workspaceID := "ws_consistent"
		token := "sk_test_token"

		// Encrypt and decrypt multiple times
		for i := 0; i < 5; i++ {
			encrypted, err := we.EncryptToken(workspaceID, token)
			require.NoError(t, err)

			decrypted, err := we.DecryptToken(workspaceID, encrypted)
			require.NoError(t, err)
			require.Equal(t, token, decrypted)
		}
	})

	t.Run("different master keys produce different encryptions", func(t *testing.T) {
		masterKey1 := make([]byte, 32)
		_, err := rand.Read(masterKey1)
		require.NoError(t, err)

		masterKey2 := make([]byte, 32)
		_, err = rand.Read(masterKey2)
		require.NoError(t, err)

		we1, err := encryption.NewWorkspaceEncryption(masterKey1)
		require.NoError(t, err)

		we2, err := encryption.NewWorkspaceEncryption(masterKey2)
		require.NoError(t, err)

		workspaceID := "ws_test"
		token := "sk_test_token"

		encrypted1, err := we1.EncryptToken(workspaceID, token)
		require.NoError(t, err)

		encrypted2, err := we2.EncryptToken(workspaceID, token)
		require.NoError(t, err)

		// Different master keys should produce different ciphertexts
		require.NotEqual(t, encrypted1, encrypted2)

		// Cannot decrypt with wrong master key
		_, err = we2.DecryptToken(workspaceID, encrypted1)
		require.Error(t, err)

		_, err = we1.DecryptToken(workspaceID, encrypted2)
		require.Error(t, err)
	})
}
