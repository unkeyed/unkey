package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

// TestKeyLifecycle_MultipleEncryptionsUseSameKey verifies that multiple
// encryptions within the same keyring reuse the same DEK until rotation.
func TestKeyLifecycle_MultipleEncryptionsUseSameKey(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-reuse"

	var keyIDs []string
	for i := 0; i < 10; i++ {
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    fmt.Sprintf("data-%d", i),
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)
		keyIDs = append(keyIDs, res.Msg.GetKeyId())
	}

	// All encryptions should use the same key
	for i := 1; i < len(keyIDs); i++ {
		require.Equal(t, keyIDs[0], keyIDs[i],
			"encryption %d used different key than encryption 0", i)
	}
}

// TestKeyLifecycle_DifferentKeyringsUseDifferentKeys verifies that
// different keyrings use different DEKs.
func TestKeyLifecycle_DifferentKeyringsUseDifferentKeys(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	keyrings := []string{"keyring-a", "keyring-b", "keyring-c"}
	keyIDs := make(map[string]string)

	for _, kr := range keyrings {
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: kr,
			Data:    "test-data",
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		res, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)
		keyIDs[kr] = res.Msg.GetKeyId()
	}

	// All keyrings should have different keys
	require.NotEqual(t, keyIDs["keyring-a"], keyIDs["keyring-b"])
	require.NotEqual(t, keyIDs["keyring-b"], keyIDs["keyring-c"])
	require.NotEqual(t, keyIDs["keyring-a"], keyIDs["keyring-c"])
}

// TestKeyLifecycle_ReEncryptPreservesData verifies that re-encryption
// preserves the original plaintext.
func TestKeyLifecycle_ReEncryptPreservesData(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-reencrypt"
	originalData := "sensitive-information-to-preserve"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    originalData,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Re-encrypt
	reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   keyring,
		Encrypted: encRes.Msg.GetEncrypted(),
	})
	reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	reencRes, err := service.ReEncrypt(ctx, reencReq)
	require.NoError(t, err)

	// Decrypt the re-encrypted data
	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: reencRes.Msg.GetEncrypted(),
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decRes, err := service.Decrypt(ctx, decReq)
	require.NoError(t, err)
	require.Equal(t, originalData, decRes.Msg.GetPlaintext())
}

// TestKeyLifecycle_ReEncryptProducesDifferentCiphertext verifies that
// re-encryption produces a different ciphertext (due to new nonce).
func TestKeyLifecycle_ReEncryptProducesDifferentCiphertext(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-reencrypt-diff"
	data := "test-data"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	originalEncrypted := encRes.Msg.GetEncrypted()

	// Re-encrypt
	reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   keyring,
		Encrypted: originalEncrypted,
	})
	reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	reencRes, err := service.ReEncrypt(ctx, reencReq)
	require.NoError(t, err)

	// Ciphertexts should be different (different nonce)
	require.NotEqual(t, originalEncrypted, reencRes.Msg.GetEncrypted(),
		"re-encrypted data should have different ciphertext")
}

// TestKeyLifecycle_OldCiphertextStillDecryptable verifies that after
// re-encryption, the original ciphertext is still decryptable.
func TestKeyLifecycle_OldCiphertextStillDecryptable(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-old-decrypt"
	data := "data-to-reencrypt"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	originalEncrypted := encRes.Msg.GetEncrypted()

	// Re-encrypt
	reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   keyring,
		Encrypted: originalEncrypted,
	})
	reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	_, err = service.ReEncrypt(ctx, reencReq)
	require.NoError(t, err)

	// Original ciphertext should still be decryptable
	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: originalEncrypted,
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decRes, err := service.Decrypt(ctx, decReq)
	require.NoError(t, err)
	require.Equal(t, data, decRes.Msg.GetPlaintext())
}

// TestKeyLifecycle_MultipleReEncryptions verifies that data can be
// re-encrypted multiple times without data loss.
func TestKeyLifecycle_MultipleReEncryptions(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()
	keyring := "test-keyring-multi-reencrypt"
	data := "data-for-multiple-reencryptions"

	// Encrypt
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: keyring,
		Data:    data,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)
	encrypted := encRes.Msg.GetEncrypted()

	// Re-encrypt multiple times
	for i := 0; i < 5; i++ {
		reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
			Keyring:   keyring,
			Encrypted: encrypted,
		})
		reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		reencRes, err := service.ReEncrypt(ctx, reencReq)
		require.NoError(t, err, "re-encryption %d failed", i)
		encrypted = reencRes.Msg.GetEncrypted()
	}

	// Decrypt final result
	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   keyring,
		Encrypted: encrypted,
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decRes, err := service.Decrypt(ctx, decReq)
	require.NoError(t, err)
	require.Equal(t, data, decRes.Msg.GetPlaintext())
}

// TestKeyLifecycle_CrossKeyringDecryptFails verifies that encrypted data
// cannot be decrypted with a different keyring.
func TestKeyLifecycle_CrossKeyringDecryptFails(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// Encrypt with keyring A
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "keyring-source",
		Data:    "secret-data",
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Try to decrypt with keyring B
	decReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "keyring-target",
		Encrypted: encRes.Msg.GetEncrypted(),
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	_, err = service.Decrypt(ctx, decReq)
	require.Error(t, err, "decryption with wrong keyring should fail")
}

// TestKeyLifecycle_CrossKeyringReEncryptFails verifies that encrypted data
// cannot be re-encrypted with a different keyring.
func TestKeyLifecycle_CrossKeyringReEncryptFails(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// Encrypt with keyring A
	encReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "keyring-encrypt",
		Data:    "secret-data",
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.Encrypt(ctx, encReq)
	require.NoError(t, err)

	// Try to re-encrypt with keyring B
	reencReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "keyring-reencrypt",
		Encrypted: encRes.Msg.GetEncrypted(),
	})
	reencReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	_, err = service.ReEncrypt(ctx, reencReq)
	require.Error(t, err, "re-encryption with wrong keyring should fail")
}
