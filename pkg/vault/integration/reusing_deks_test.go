package integration_test

import (
	"context"
	"testing"

	"fmt"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/vault/keys"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"time"
)

// When encrypting multiple secrets with the same keyring, the same DEK should be reused for all of them.
func TestReuseDEKsForSameKeyring(t *testing.T) {

	logger := logging.NewNoop()

	s3 := containers.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          fmt.Sprintf("%d", time.Now().UnixMilli()),
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

	deks := map[string]bool{}

	for range 10 {
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: "keyring",
			Data:    uid.New(uid.TestPrefix),
		})
		require.NoError(t, encryptErr)
		deks[res.GetKeyId()] = true
	}

	require.Len(t, deks, 1)

}

// When encrypting multiple secrets with different keyrings, a different DEK should be used for each keyring.
func TestIndividualDEKsPerKeyring(t *testing.T) {

	logger := logging.NewNoop()

	s3 := containers.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          fmt.Sprintf("%d", time.Now().UnixMilli()),
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

	deks := map[string]bool{}

	for range 10 {
		res, encryptErr := v.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: uid.New(uid.TestPrefix),
			Data:    uid.New(uid.TestPrefix),
		})
		require.NoError(t, encryptErr)
		deks[res.GetKeyId()] = true
	}

	require.Len(t, deks, 10)

}
