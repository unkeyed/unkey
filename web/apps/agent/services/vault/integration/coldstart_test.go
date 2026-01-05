package integration_test

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/testutils/containers"
	"github.com/unkeyed/unkey/svc/agent/services/vault"
	"github.com/unkeyed/unkey/svc/agent/services/vault/keys"
	"github.com/unkeyed/unkey/svc/agent/services/vault/storage"
)

// This scenario tests the cold start of the vault service.
// There are no keys in the storage and a few users are starting to use it

func Test_ColdStart(t *testing.T) {

	logger := logging.New(nil).Level(zerolog.ErrorLevel)

	s3 := containers.NewS3(t)
	defer s3.Stop()

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          "vault",
		S3AccessKeyId:     s3.AccessKeyId,
		S3AccessKeySecret: s3.AccessKeySecret,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	v, err := vault.New(vault.Config{
		Storage:    storage,
		Logger:     logger,
		MasterKeys: []string{masterKey},
	})
	require.NoError(t, err)

	ctx := context.Background()

	// Alice encrypts a secret
	aliceData := "alice secret"
	aliceEncryptionRes, err := v.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: "alice",
		Data:    aliceData,
	})
	require.NoError(t, err)

	// Bob encrypts a secret
	bobData := "bob secret"
	bobEncryptionRes, err := v.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: "bob",
		Data:    bobData,
	})
	require.NoError(t, err)

	// Alice decrypts her secret
	aliceDecryptionRes, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   "alice",
		Encrypted: aliceEncryptionRes.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, aliceData, aliceDecryptionRes.Plaintext)

	// Bob reencrypts his secret

	_, err = v.CreateDEK(ctx, &vaultv1.CreateDEKRequest{
		Keyring: "bob",
	})
	require.NoError(t, err)
	bobReencryptionRes, err := v.ReEncrypt(ctx, &vaultv1.ReEncryptRequest{
		Keyring:   "bob",
		Encrypted: bobEncryptionRes.Encrypted,
	})
	require.NoError(t, err)

	// Bob decrypts his secret
	bobDecryptionRes, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   "bob",
		Encrypted: bobReencryptionRes.Encrypted,
	})
	require.NoError(t, err)
	require.Equal(t, bobData, bobDecryptionRes.Plaintext)
	// expect the key to be different
	require.NotEqual(t, bobEncryptionRes.KeyId, bobReencryptionRes.KeyId)

}
