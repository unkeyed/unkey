package vault

import (
	"context"
	"fmt"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestEncrypt_RefreshesStaleKeyAfterRotation(t *testing.T) {
	clk := clock.NewTestClock()
	service := setupTestServiceWithClock(t, clk)
	keyring := uid.New(uid.WorkspacePrefix)
	req := connect.NewRequest(&vaultv1.EncryptRequest{Keyring: keyring, Data: "secret"})
	req.Header().Set("Authorization", "Bearer "+service.bearer)
	original, err := service.Encrypt(t.Context(), req)
	require.NoError(t, err)
	keyIDNew, err := service.CreateDEK(t.Context(), keyring)
	require.NoError(t, err)
	require.NotEqual(t, original.Msg.GetKeyId(), keyIDNew)

	fresh, err := service.Encrypt(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, original.Msg.GetKeyId(), fresh.Msg.GetKeyId())

	clk.Tick(time.Hour)
	stale, err := service.Encrypt(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, original.Msg.GetKeyId(), stale.Msg.GetKeyId())

	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		refreshed, err := service.Encrypt(t.Context(), req)
		require.NoError(collect, err)
		require.Equal(collect, keyIDNew, refreshed.Msg.GetKeyId())
	}, 5*time.Second, 10*time.Millisecond)
}

func TestEncrypt_WithValidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// Create request with proper Authorization header
	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})
	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	res, err := service.Encrypt(ctx, req)
	require.NoError(t, err)
	require.NotEmpty(t, res.Msg.GetEncrypted())
	require.NotEmpty(t, res.Msg.GetKeyId())
}

func TestEncrypt_WithoutAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	// Create request WITHOUT Authorization header
	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})

	_, err := service.Encrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestEncrypt_WithInvalidAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	_, err := service.Encrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestEncrypt_WithEmptyAuth(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})
	req.Header().Set("Authorization", "Bearer ")

	_, err := service.Encrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestEncrypt_WithInvalidScheme(t *testing.T) {
	service := setupTestService(t)
	ctx := context.Background()

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test-keyring",
		Data:    "secret data",
	})
	req.Header().Set("Authorization", "Basic test-token")

	_, err := service.Encrypt(ctx, req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
