package integration

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/apps/vault/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/apps/vault/pkg/keys"
	"github.com/unkeyed/unkey/apps/vault/pkg/logging"
	"github.com/unkeyed/unkey/apps/vault/pkg/service"
	"github.com/unkeyed/unkey/apps/vault/pkg/storage"
)

// This scenario tests the re-encryption of a secret
func TestMigrateDeks(t *testing.T) {

	logger := logging.New(nil)



	data := make(map[string]string)



	storage, err := storage.NewMemory(storage.MemoryConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	_,masterKeyOld, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	vault, err := service.New(service.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKeyOld},
	})
	require.NoError(t, err)

	ctx := context.Background()




	// Seed some DEKs
	for i := 0; i<10; i++{
		_, err := vault.CreateDEK(ctx, &vaultv1.CreateDEKRequest{
			Shard: "shard",
		})
		require.NoError(t, err)

		buf := make([]byte, 32)
		_, err = rand.Read(buf)
		d := string(buf)
		require.NoError(t, err)
		res, err := vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Shard: "shard",
			Data: string(d),
		})
		require.NoError(t, err)
		data[d] = res.Encrypted
	}


	// Simulate Restart

	_,masterKeyNew, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	vault, err = service.New(service.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKeyOld,masterKeyNew},
	})
	require.NoError(t, err)

	
err = vault.RollDeks(ctx)
	require.NoError(t, err)


	// Check each piece of data can be decrypted
	for d, e := range data{
		res, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Shard: "shard",
			Encrypted: e,
		})
		require.NoError(t, err)
		require.Equal(t, d, res.Plaintext)
	}
// Simualte another restart, removing the old master key

	vault, err = service.New(service.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKeyNew},
	})
	require.NoError(t, err)


	// Check each piece of data can be decrypted
	for d, e := range data{
		res, err := vault.Decrypt(ctx, &vaultv1.DecryptRequest{
			Shard: "shard",
			Encrypted: e,
		})
		require.NoError(t, err)
		require.Equal(t, d, res.Plaintext)
	}


}