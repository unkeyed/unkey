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
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/vault/internal/keys"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
)

// This scenario tests the re-encryption of a secret.
func TestReEncrypt(t *testing.T) {

	logger := logging.NewNoop()

	s3 := containers.S3(t)

	storage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          "vault",
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

	for i := 1; i < 9; i++ {

		dataSize := int(math.Pow(8, float64(i)))
		t.Run(fmt.Sprintf("with %d bytes", dataSize), func(t *testing.T) {

			keyring := uid.New("test")
			buf := make([]byte, dataSize)
			_, err := rand.Read(buf)
			require.NoError(t, err)

			data := string(buf)

			enc, err := v.Encrypt(ctx, connect.NewRequest(&vaultv1.EncryptRequest{
				Keyring: keyring,
				Data:    data,
			}))
			require.NoError(t, err)

			deks := []string{}
			for range 10 {
				dekID, createDekErr := v.CreateDEK(ctx, keyring)
				require.NoError(t, createDekErr)
				require.NotContains(t, deks, dekID)
				deks = append(deks, dekID)
				_, err = v.ReEncrypt(ctx, connect.NewRequest(&vaultv1.ReEncryptRequest{
					Keyring:   keyring,
					Encrypted: enc.Msg.GetEncrypted(),
				}))
				require.NoError(t, err)
			}

			dec, err := v.Decrypt(ctx, connect.NewRequest(&vaultv1.DecryptRequest{
				Keyring:   keyring,
				Encrypted: enc.Msg.GetEncrypted(),
			}))
			require.NoError(t, err)
			require.Equal(t, data, dec.Msg.GetPlaintext())
		})

	}

}
