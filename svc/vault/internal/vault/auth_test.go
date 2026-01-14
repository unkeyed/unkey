package vault

import (
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
)

func TestAuthenticate_Success(t *testing.T) {
	service := setupTestService(t)

	// Mock request with valid Authorization header
	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test",
		Data:    "test",
	})
	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", service.bearer))

	err := service.authenticate(req)
	require.NoError(t, err)
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	service := setupTestService(t)

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test",
		Data:    "test",
	})
	// No Authorization header set

	err := service.authenticate(req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAuthenticate_InvalidScheme(t *testing.T) {
	service := setupTestService(t)

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test",
		Data:    "test",
	})
	req.Header().Set("Authorization", "Basic "+service.bearer)

	err := service.authenticate(req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAuthenticate_EmptyToken(t *testing.T) {
	service := setupTestService(t)

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test",
		Data:    "test",
	})
	req.Header().Set("Authorization", "Bearer ")

	err := service.authenticate(req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	service := setupTestService(t)

	req := connect.NewRequest(&vaultv1.EncryptRequest{
		Keyring: "test",
		Data:    "test",
	})
	req.Header().Set("Authorization", "Bearer wrong-token")

	err := service.authenticate(req)
	require.Error(t, err)
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
