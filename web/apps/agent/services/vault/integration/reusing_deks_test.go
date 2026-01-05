package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/testutils/containers"
	"github.com/unkeyed/unkey/svc/agent/services/vault"
	"github.com/unkeyed/unkey/svc/agent/services/vault/keys"
	"github.com/unkeyed/unkey/svc/agent/services/vault/storage"
)

// When encrypting multiple secrets with the same keyring, the same DEK should be reused for all of them.
func TestReuseDEKsForSameKeyring(t *testing.T) {

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

	deks := map[string]bool{}

	for range 10 {
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: "keyring",
			Data:    uuid.NewString(),
		})
		require.NoError(t, encryptErr)
		deks[res.KeyId] = true
	}

	require.Len(t, deks, 1)

}

// When encrypting multiple secrets with different keyrings, a different DEK should be used for each keyring.
func TestIndividualDEKsPerKeyring(t *testing.T) {

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

	deks := map[string]bool{}

	for range 10 {
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: uuid.NewString(),
			Data:    uuid.NewString(),
		})
		require.NoError(t, encryptErr)
		deks[res.KeyId] = true
	}

	require.Len(t, deks, 10)

}
