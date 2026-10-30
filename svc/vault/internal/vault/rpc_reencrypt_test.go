package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestReEncryptBulk_WithValidAuth(t *testing.T) {
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
	id := uid.New(uid.TestPrefix)
	reencryptReq := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			id: {Keyring: "test-keyring", Encrypted: encryptRes.Msg.GetEncrypted()},
		},
	})
	reencryptReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	reencryptRes, err := service.ReEncryptBulk(ctx, reencryptReq)
	require.NoError(t, err)
	result := reencryptRes.Msg.GetItems()[id]
	require.NotNil(t, result)
	require.NotEmpty(t, result.GetEncrypted())
	require.NotEmpty(t, result.GetKeyId())
	// Re-encryption might use the same key if it's already the latest
}

func TestReEncryptBulk_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			uid.New(uid.TestPrefix): {Keyring: "test-keyring", Encrypted: "some-encrypted-data"},
		},
	})

	_, err := service.ReEncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncryptBulk_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			uid.New(uid.TestPrefix): {Keyring: "test-keyring", Encrypted: "some-encrypted-data"},
		},
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.ReEncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncryptBulk_WithEmptyAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			uid.New(uid.TestPrefix): {Keyring: "test-keyring", Encrypted: "some-encrypted-data"},
		},
	})
	req.Header().Set("Authorization", "Bearer ")

	_, err := service.ReEncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestReEncryptBulk_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.ReEncryptBulkRequest{
		Items: map[string]*vaultv1.DecryptRequest{
			uid.New(uid.TestPrefix): {Keyring: "test-keyring", Encrypted: "some-encrypted-data"},
		},
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.ReEncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
