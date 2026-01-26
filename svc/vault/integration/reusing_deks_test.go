package integration_test

import (
	"context"
	"testing"

	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/keys"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
)

// TestReuseDEKsForSameKeyring verifies that multiple encrypts with the same keyring
// reuse the same DEK.
//
// This is important for efficiency - we don't want to create a new DEK for every
// encrypt operation. All secrets within a keyring should use the same DEK until
// it is explicitly rotated.
func TestReuseDEKsForSameKeyring(t *testing.T) {

	logger := logging.NewNoop()

	s3 := dockertest.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          fmt.Sprintf("%d", time.Now().UnixMilli()),
		S3AccessKeyID:     s3.AccessKeyID,
		S3AccessKeySecret: s3.SecretAccessKey,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	bearer := "bearer"

	v, err := vault.New(vault.Config{
		Storage:     storage,
		Logger:      logger,
		MasterKeys:  []string{masterKey},
		BearerToken: bearer,
	})
	require.NoError(t, err)

	ctx := context.Background()

	deks := map[string]bool{}

	for range 10 {
		req := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: "keyring",
			Data:    uid.New(uid.TestPrefix),
		})
		req.Header().Add("Authorization", fmt.Sprintf("Bearer %s", bearer))
		res, encryptErr := v.Encrypt(ctx, req)
		require.NoError(t, encryptErr)
		deks[res.Msg.GetKeyId()] = true
	}

	require.Len(t, deks, 1)

}

// TestIndividualDEKsPerKeyring verifies that different keyrings use different DEKs.
//
// This provides tenant isolation - each keyring (typically representing a workspace
// or tenant) gets its own encryption key. Compromise of one keyring's DEK does not
// affect other keyrings.
func TestIndividualDEKsPerKeyring(t *testing.T) {

	logger := logging.NewNoop()

	s3 := dockertest.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          fmt.Sprintf("%d", time.Now().UnixMilli()),
		S3AccessKeyID:     s3.AccessKeyID,
		S3AccessKeySecret: s3.SecretAccessKey,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)
	bearer := "bearer"

	v, err := vault.New(vault.Config{
		Storage:     storage,
		Logger:      logger,
		MasterKeys:  []string{masterKey},
		BearerToken: bearer,
	})
	require.NoError(t, err)

	ctx := context.Background()

	deks := map[string]bool{}

	for range 10 {
		req := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: uid.New(uid.TestPrefix),
			Data:    uid.New(uid.TestPrefix),
		})
		req.Header().Add("Authorization", fmt.Sprintf("Bearer %s", bearer))
		res, encryptErr := v.Encrypt(ctx, req)
		require.NoError(t, encryptErr)
		deks[res.Msg.GetKeyId()] = true
	}

	require.Len(t, deks, 10)

}
