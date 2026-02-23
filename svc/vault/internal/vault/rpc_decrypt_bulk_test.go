package vault

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func TestDecryptBulk_Roundtrip(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	plaintexts := map[string]string{
		"key-1": "secret one",
		"key-2": "secret two",
		"key-3": "secret three",
	}

	// Encrypt each individually first
	encrypted := make(map[string]*vaultv1.DecryptBulkRequestItem, len(plaintexts))
	for id, data := range plaintexts {
		encReq := connect.NewRequest(&vaultv1.EncryptRequest{
			Keyring: "test-keyring",
			Data:    data,
		})
		encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

		encRes, err := service.Encrypt(ctx, encReq)
		require.NoError(t, err)
		encrypted[id] = &vaultv1.DecryptBulkRequestItem{
			Encrypted: encRes.Msg.GetEncrypted(),
		}
	}

	// Decrypt in bulk
	decReq := connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "test-keyring",
		Items:   encrypted,
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decRes, err := service.DecryptBulk(ctx, decReq)
	require.NoError(t, err)
	require.Len(t, decRes.Msg.GetItems(), 3)

	// Verify all plaintexts match
	for id, expected := range plaintexts {
		item, ok := decRes.Msg.GetItems()[id]
		require.True(t, ok, "missing response key: %s", id)
		require.Equal(t, expected, item.GetPlaintext())
	}
}

func TestDecryptBulk_EmptyItems(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "test-keyring",
		Items:   map[string]*vaultv1.DecryptBulkRequestItem{},
	})
	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	res, err := service.DecryptBulk(ctx, req)
	require.NoError(t, err)
	require.Empty(t, res.Msg.GetItems())
}

func TestDecryptBulk_ResponseKeysMatchRequestKeys(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// Encrypt via bulk
	encReq := connect.NewRequest(&vaultv1.EncryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.EncryptBulkRequestItem{
			"alpha": {Data: "data-a"},
			"beta":  {Data: "data-b"},
		},
	})
	encReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	encRes, err := service.EncryptBulk(ctx, encReq)
	require.NoError(t, err)

	// Build decrypt request with same keys
	decItems := make(map[string]*vaultv1.DecryptBulkRequestItem, len(encRes.Msg.GetItems()))
	for id, item := range encRes.Msg.GetItems() {
		decItems[id] = &vaultv1.DecryptBulkRequestItem{
			Encrypted: item.GetEncrypted(),
		}
	}

	decReq := connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "test-keyring",
		Items:   decItems,
	})
	decReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	decRes, err := service.DecryptBulk(ctx, decReq)
	require.NoError(t, err)

	// Response keys must exactly match request keys
	require.Len(t, decRes.Msg.GetItems(), len(decItems))
	for id := range decItems {
		_, ok := decRes.Msg.GetItems()[id]
		require.True(t, ok, "missing response key: %s", id)
	}
}

func TestDecryptBulk_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.DecryptBulkRequestItem{
			"key-1": {Encrypted: "some-data"},
		},
	})

	_, err := service.DecryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecryptBulk_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.DecryptBulkRequestItem{
			"key-1": {Encrypted: "some-data"},
		},
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.DecryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestDecryptBulk_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.DecryptBulkRequest{
		Keyring: "test-keyring",
		Items: map[string]*vaultv1.DecryptBulkRequestItem{
			"key-1": {Encrypted: "some-data"},
		},
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.DecryptBulk(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
