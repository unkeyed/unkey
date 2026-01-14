package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func TestReEncrypt_WithValidAuth(t *testing.T) {
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

	// Then re-encrypt it
	reencryptReq := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: encryptRes.Msg.GetEncrypted(),
	})
	reencryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	reencryptRes, err := service.ReEncrypt(ctx, reencryptReq)
	require.NoError(t, err)
	require.NotEmpty(t, reencryptRes.Msg.GetEncrypted())
	require.NotEmpty(t, reencryptRes.Msg.GetKeyId())
	// Re-encryption might use the same key if it's already the latest
}

func TestReEncrypt_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncrypt_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncrypt_WithEmptyAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Bearer ")

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncrypt_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptRequest{
		Keyring:   "test-keyring",
		Encrypted: "some-encrypted-data",
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.ReEncrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
