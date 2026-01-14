package integration_test

import (
	"context"
	"crypto/rand"
	"testing"

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

// TestMigrateDeks verifies that DEKs remain decryptable after KEK rotation.
//
// This scenario tests the master key migration process:
//  1. Encrypt data with the old master key
//  2. Simulate a service restart with a new master key (old key kept for decryption)
//  3. Verify all existing encrypted data can still be decrypted
//
// This is critical for key rotation - users must never lose access to their secrets
// when the master key is rotated.
func TestMigrateDeks(t *testing.T) {

	logger := logging.NewNoop()
	data := make(map[string]string)
	bearerToken := "integration-test-token"
	s3 := dockertest.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          "test",
		S3AccessKeyID:     s3.AccessKeyID,
		S3AccessKeySecret: s3.SecretAccessKey,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKeyOld, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	v, err := vault.New(vault.Config{
		Storage:     storage,
		Logger:      logger,
		MasterKeys:  []string{masterKeyOld},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	ctx := context.Background()

	keyring := uid.New("test")
	// Seed some DEKs
	for range 10 {

		_, err = v.CreateDEK(ctx, keyring)
		require.NoError(t, err)

		buf := make([]byte, 32)
		_, err = rand.Read(buf)
		d := string(buf)
		require.NoError(t, err)
		encryptReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: keyring,
			Data:    d,
		})
		encryptReq.Header().Set("Authorization", "Bearer "+bearerToken)
		res, encryptErr := v.Encrypt(ctx, encryptReq)
		require.NoError(t, encryptErr)
		data[d] = res.Msg.GetEncrypted()
	}

	// Simulate Restart

	_, masterKeyNew, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	v, err = vault.New(vault.Config{
		Storage:     storage,
		Logger:      logger,
		MasterKeys:  []string{masterKeyNew, masterKeyOld},
		BearerToken: bearerToken,
	})
	require.NoError(t, err)

	// Check each piece of data can be decrypted
	for d, e := range data {
		decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   keyring,
			Encrypted: e,
		})
		decryptReq.Header().Set("Authorization", "Bearer "+bearerToken)
		res, decryptErr := v.Decrypt(ctx, decryptReq)
		require.NoError(t, decryptErr)
		require.Equal(t, d, res.Msg.GetPlaintext())
	}

}
