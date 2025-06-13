package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/keys"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

// This scenario tests the cold start of the vault service.
// There are no keys in the storage and a few users are starting to use it

func Test_ColdStart(t *testing.T) {

	c := containers.New(t)
	s3 := c.RunS3(t)

	logger := logging.NewNoop()

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          uid.New("", 8),
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
		Encrypted: aliceEncryptionRes.GetEncrypted(),
	})
	require.NoError(t, err)
	require.Equal(t, aliceData, aliceDecryptionRes.GetPlaintext())

	// Bob reencrypts his secret

	_, err = v.CreateDEK(ctx, &vaultv1.CreateDEKRequest{
		Keyring: "bob",
	})
	require.NoError(t, err)
	bobReencryptionRes, err := v.ReEncrypt(ctx, &vaultv1.ReEncryptRequest{
		Keyring:   "bob",
		Encrypted: bobEncryptionRes.GetEncrypted(),
	})
	require.NoError(t, err)

	// Bob decrypts his secret
	bobDecryptionRes, err := v.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   "bob",
		Encrypted: bobReencryptionRes.GetEncrypted(),
	})
	require.NoError(t, err)
	require.Equal(t, bobData, bobDecryptionRes.GetPlaintext())
	// expect the key to be different
	require.NotEqual(t, bobEncryptionRes.GetKeyId(), bobReencryptionRes.GetKeyId())

}
