package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func TestEncryptBulk_Roundtrip(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	items := map[string]*vaultv1.EncryptBulkRequestItem{
		"key-1": {Data: "secret one"},
		"key-2": {Data: "secret two"},
		"key-3": {Data: "secret three"},
	}

	encReq := connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "test-keyring",
		Items:   items,
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.EncryptBulk(ctx, encReq)
	require.NoError(t, err)
	require.Len(t, encRes.Msg.GetItems(), 3)

	// Verify all keys are present and encrypted values are non-empty
	for id := range items {
		item, ok := encRes.Msg.GetItems()[id]
		require.True(t, ok, "missing response key: %s", id)
		require.NotEmpty(t, item.GetEncrypted())
		require.NotEmpty(t, item.GetKeyId())
	}

	// Decrypt each individually and verify plaintext matches
	for id, original := range items {
		decReq := connect.NewRequest(&vaultv1.DecryptRequest{
			Keyring:   "test-keyring",
			Encrypted: encRes.Msg.GetItems()[id].GetEncrypted(),
		})
		decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		decRes, err := service.Decrypt(ctx, decReq)
		require.NoError(t, err)
		require.Equal(t, original.GetData(), decRes.Msg.GetPlaintext())
	}
}

func TestEncryptBulk_EmptyItems(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "test-keyring",
		Items:   map[string]*vaultv1.EncryptBulkRequestItem{},
	})
	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	res, err := service.EncryptBulk(ctx, req)
	require.NoError(t, err)
	require.Empty(t, res.Msg.GetItems())
}

func TestEncryptBulk_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.EncryptBulkRequestItem{
			"key-1": {Data: "secret"},
		},
	})

	_, err := service.EncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestEncryptBulk_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.EncryptBulkRequestItem{
			"key-1": {Data: "secret"},
		},
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.EncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestEncryptBulk_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.EncryptBulkRequestItem{
			"key-1": {Data: "secret"},
		},
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.EncryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
