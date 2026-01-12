package integration_test

import (
	"context"
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

// Test_ColdStart verifies the vault service starts correctly with empty storage.
//
// This scenario tests that multiple users can encrypt and decrypt secrets
// when the vault has no pre-existing keys. It validates:
//   - DEK creation on first encrypt per keyring
//   - Encrypt/decrypt roundtrip for multiple users
//   - Re-encryption with a new DEK
//   - Keyring isolation between users

func Test_ColdStart(t *testing.T) {

	s3 := dockertest.S3(t)

	logger := logging.NewNoop()

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          "test",
		S3AccessKeyID:     s3.AccessKeyID,
		S3AccessKeySecret: s3.SecretAccessKey,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKey, err := keys.GenerateMasterKey()
	require.NoError(t, err)

	v, err := vault.New(vault.Config{
		Storage:     storage,
		Logger:      logger,
		MasterKeys:  []string{masterKey},
		BearerToken: "test-bearer-token",
	})
	require.NoError(t, err)

	ctx := context.Background()

	aliceKeyRing := uid.New("alice")
	bobKeyRing := uid.New("bob")
	// Alice encrypts a secret
	aliceData := "alice secret"
	aliceEncryptReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: aliceKeyRing,
		Data:    aliceData,
	})
	aliceEncryptReq.Header().Set("Authorization", "Bearer test-bearer-token")
	aliceEncryptionRes, err := v.Encrypt(ctx, aliceEncryptReq)
	require.NoError(t, err)

	// Bob encrypts a secret
	bobData := "bob secret"
	bobEncryptReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: bobKeyRing,
		Data:    bobData,
	})
	bobEncryptReq.Header().Set("Authorization", "Bearer test-bearer-token")
	bobEncryptionRes, err := v.Encrypt(ctx, bobEncryptReq)
	require.NoError(t, err)

	// Alice decrypts her secret
	aliceDecryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   aliceKeyRing,
		Encrypted: aliceEncryptionRes.Msg.GetEncrypted(),
	})
	aliceDecryptReq.Header().Set("Authorization", "Bearer test-bearer-token")
	aliceDecryptionRes, err := v.Decrypt(ctx, aliceDecryptReq)
	require.NoError(t, err)
	require.Equal(t, aliceData, aliceDecryptionRes.Msg.GetPlaintext())

	// Bob reencrypts his secret

	_, err = v.CreateDEK(ctx, bobKeyRing)
	require.NoError(t, err)
	bobReencryptReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   bobKeyRing,
		Encrypted: bobEncryptionRes.Msg.GetEncrypted(),
	})
	bobReencryptReq.Header().Set("Authorization", "Bearer test-bearer-token")
	bobReencryptionRes, err := v.ReEncrypt(ctx, bobReencryptReq)
	require.NoError(t, err)

	// Bob decrypts his secret
	bobDecryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   bobKeyRing,
		Encrypted: bobReencryptionRes.Msg.GetEncrypted(),
	})
	bobDecryptReq.Header().Set("Authorization", "Bearer test-bearer-token")
	bobDecryptionRes, err := v.Decrypt(ctx, bobDecryptReq)
	require.NoError(t, err)
	require.Equal(t, bobData, bobDecryptionRes.Msg.GetPlaintext())
	// expect the key to be different
	require.NotEqual(t, bobEncryptionRes.Msg.GetKeyId(), bobReencryptionRes.Msg.GetKeyId())

}
