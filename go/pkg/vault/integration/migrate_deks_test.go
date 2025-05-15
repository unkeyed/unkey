package integration_test

import (
	"context"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/keys"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

// This scenario tests the re-encryption of a secret.
func TestMigrateDeks(t *testing.T) {

	logger := logging.NewNoop()
	c := containers.New(t)
	data := make(map[string]string)
	s3 := c.RunS3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          "vault",
		S3AccessKeyId:     s3.AccessKeyId,
		S3AccessKeySecret: s3.AccessKeySecret,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKeyOld, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	v, err := vault.New(vault.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKeyOld},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Seed some DEKs
	for range 10 {
		_, err = v.CreateDEK(ctx, &vaultv1.CreateDEKRequest{
			Keyring: "keyring",
		})
		require.NoError(t, err)

		buf := make([]byte, 32)
		_, err = rand.Read(buf)
		d := string(buf)
		require.NoError(t, err)
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: "keyring",
			Data:    string(d),
		})
		require.NoError(t, encryptErr)
		data[d] = res.Encrypted
	}

	// Simulate Restart

	_, masterKeyNew, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	v, err = vault.New(vault.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKeyOld, masterKeyNew},
	})
	require.NoError(t, err)

	err = v.RollDeks(ctx)
	require.NoError(t, err)

	// Check each piece of data can be decrypted
	for d, e := range data {
		res, decryptErr := v.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   "keyring",
			Encrypted: e,
		})
		require.NoError(t, decryptErr)
		require.Equal(t, d, res.Plaintext)
	}
	// Simulate another restart, removing the old master key

	v, err = vault.New(vault.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKeyNew},
	})
	require.NoError(t, err)

	// Check each piece of data can be decrypted
	for d, e := range data {
		res, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
			Keyring:   "keyring",
			Encrypted: e,
		})
		require.NoError(t, err)
		require.Equal(t, d, res.Plaintext)
	}

}
