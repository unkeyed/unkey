package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func TestDecrypt_WithValidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// First encrypt some data
	encryptReq := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})
	encryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encryptRes, err := service.Encrypt(ctx, encryptReq)
	require.NoError(t, err)

	// Then decrypt it
	decryptReq := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: encryptRes.Msg.GetEncrypted(),
	})
	decryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decryptRes, err := service.Decrypt(ctx, decryptReq)
	require.NoError(t, err)
	require.Equal(t, "secret data", decryptRes.Msg.GetPlaintext())
}

func TestDecrypt_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecrypt_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecrypt_WithEmptyAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer ")

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecrypt_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.Decrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
