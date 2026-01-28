package integration_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
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

// TestReEncrypt verifies that re-encryption works correctly across varying data sizes.
//
// This test encrypts data of increasing sizes (8^1 to 8^8 bytes), then performs
// multiple DEK rotations and verifies the original encrypted data can still be
// decrypted. This ensures:
//   - Large data is handled correctly
//   - Re-encryption with new DEKs doesn't lose data
//   - Old ciphertexts remain valid after DEK rotation
func TestReEncrypt(t *testing.T) {

	logger := logging.NewNoop()

	s3 := dockertest.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.URL,
		S3Bucket:          "vault",
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

	for i := 1; i < 9; i++ {

		dataSize := int(math.Pow(8, float64(i)))
		t.Run(fmt.Sprintf("with %d bytes", dataSize), func(t *testing.T) {

			keyring := uid.New("test")
			buf := make([]byte, dataSize)
			_, err := rand.Read(buf)
			require.NoError(t, err)

			data := string(buf)

			encReq := connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    data,
			})
			encReq.Header().Add("Authorization", fmt.Sprintf("Bearer %s", bearer))
			enc, err := v.Encrypt(ctx, encReq)
			require.NoError(t, err)

			deks := []string{}
			for range 10 {
				dekID, createDekErr := v.CreateDEK(ctx, keyring)
				require.NoError(t, createDekErr)
				require.NotContains(t, deks, dekID)
				deks = append(deks, dekID)
				reReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
					Keyring:   keyring,
					Encrypted: enc.Msg.GetEncrypted(),
				})
				reReq.Header().Add("Authorization", fmt.Sprintf("Bearer %s", bearer))
				_, err = v.ReEncrypt(ctx, reReq)
				require.NoError(t, err)
			}
			decReq := connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: enc.Msg.GetEncrypted(),
			})
			decReq.Header().Add("Authorization", fmt.Sprintf("Bearer %s", bearer))
			dec, err := v.Decrypt(ctx, decReq)
			require.NoError(t, err)
			require.Equal(t, data, dec.Msg.GetPlaintext())
		})

	}

}
