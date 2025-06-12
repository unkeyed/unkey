package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/keys"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
)

// When encrypting multiple secrets with the same keyring, the same DEK should be reused for all of them.
func TestReuseDEKsForSameKeyring(t *testing.T) {

	logger := logging.NewNoop()

	c := containers.New(t)
	s3 := c.RunS3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
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

	deks := map[string]bool{}

	for range 10 {
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: "keyring",
			Data:    uuid.NewString(),
		})
		require.NoError(t, encryptErr)
		deks[res.GetKeyId()] = true
	}

	require.Len(t, deks, 1)

}

// When encrypting multiple secrets with different keyrings, a different DEK should be used for each keyring.
func TestIndividualDEKsPerKeyring(t *testing.T) {

	logger := logging.NewNoop()

	c := containers.New(t)
	s3 := c.RunS3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
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

	deks := map[string]bool{}

	for range 10 {
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: uuid.NewString(),
			Data:    uuid.NewString(),
		})
		require.NoError(t, encryptErr)
		deks[res.GetKeyId()] = true
	}

	require.Len(t, deks, 10)

}
