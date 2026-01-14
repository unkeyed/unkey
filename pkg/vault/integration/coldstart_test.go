package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/vault/keys"
	"github.com/unkeyed/unkey/pkg/vault/storage"
)

// This scenario tests the cold start of the vault service.
// There are no keys in the storage and a few users are starting to use it

func Test_ColdStart(t *testing.T) {

	s3 := containers.S3(t)

	logger := logging.NewNoop()

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          "test",
		S3AccessKeyID:     s3.AccessKeyID,
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

	aliceKeyRing := uid.New("alice")
	bobKeyRing := uid.New("bob")
	// Alice encrypts a secret
	aliceData := "alice secret"
	aliceEncryptionRes, err := v.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: aliceKeyRing,
		Data:    aliceData,
	})
	require.NoError(t, err)

	// Bob encrypts a secret
	bobData := "bob secret"
	bobEncryptionRes, err := v.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: bobKeyRing,
		Data:    bobData,
	})
	require.NoError(t, err)

	// Alice decrypts her secret
	aliceDecryptionRes, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   aliceKeyRing,
		Encrypted: aliceEncryptionRes.GetEncrypted(),
	})
	require.NoError(t, err)
	require.Equal(t, aliceData, aliceDecryptionRes.GetPlaintext())

	// Bob reencrypts his secret

	_, err = v.CreateDEK(ctx, bobKeyRing)
	require.NoError(t, err)
	bobReencryptionRes, err := v.ReEncrypt(ctx, &vaultv1.ReEncryptRequest{
		Keyring:   bobKeyRing,
		Encrypted: bobEncryptionRes.GetEncrypted(),
	})
	require.NoError(t, err)

	// Bob decrypts his secret
	bobDecryptionRes, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   bobKeyRing,
		Encrypted: bobReencryptionRes.GetEncrypted(),
	})
	require.NoError(t, err)
	require.Equal(t, bobData, bobDecryptionRes.GetPlaintext())
	// expect the key to be different
	require.NotEqual(t, bobEncryptionRes.GetKeyId(), bobReencryptionRes.GetKeyId())

}
