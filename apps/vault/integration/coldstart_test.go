package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/keys"
	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
	"github.com/unkeyed/unkey/apps/vault/pkg/service"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
)

// This scenario tests the cold start of the vault service.
// There are no keys in the storage and a few users are starting to use it

func Test_ColdStart(t *testing.T) {

	logger := logging.New(nil)

	storage, err := storage.NewMemory(storage.MemoryConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	vault, err := service.New(service.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKey},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Alice encrypts a secret
	aliceData := "alice secret"
	aliceEncryptionRes, err := vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: "alice",
		Data:    aliceData,
	})
	require.NoError(t, err)

	// Bob encrypts a secret
	bobData := "bob secret"
	bobEncryptionRes, err := vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: "bob",
		Data:    bobData,
	})
	require.NoError(t, err)

	// Alice decrypts her secret
	aliceDecryptionRes, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   "alice",
		Encrypted: aliceEncryptionRes.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, aliceData, aliceDecryptionRes.Plaintext)

	// Bob reencrypts his secret

	_, err = vault.CreateDEK(ctx, &vaultv1.CreateDEKRequest{
		Keyring: "bob",
	})
	require.NoError(t, err)
	bobReencryptionRes, err := vault.ReEncrypt(ctx, &vaultv1.ReEncryptRequest{
		Keyring:   "bob",
		Encrypted: bobEncryptionRes.Encrypted,
	})
	require.NoError(t, err)

	// Bob decrypts his secret
	bobDecryptionRes, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   "bob",
		Encrypted: bobReencryptionRes.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, bobData, bobDecryptionRes.Plaintext)
	// expect the key to be different
	require.NotEqual(t, bobEncryptionRes.KeyId, bobReencryptionRes.KeyId)

}
