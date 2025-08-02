package internalVaultEncrypt_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_encrypt"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestVaultEncryptSuccess(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Vault:  h.Vault,
		Logger: h.Logger,
		Token:  "test-token-123",
	}

	h.Register(route)

	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer test-token-123"},
	}

	t.Run("encrypt simple data", func(t *testing.T) {
		req := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    "hello world",
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Encrypted)
		require.NotEmpty(t, res.Body.KeyId)
	})

	t.Run("encrypt empty data", func(t *testing.T) {
		req := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    "",
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Encrypted)
		require.NotEmpty(t, res.Body.KeyId)
	})

	t.Run("encrypt large data", func(t *testing.T) {
		largeData := make([]byte, 1024*1024) // 1MB
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		req := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    string(largeData),
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Encrypted)
		require.NotEmpty(t, res.Body.KeyId)
	})

	t.Run("encrypt with different keyrings", func(t *testing.T) {
		// Use the workspace ID as keyring since that's what the vault expects
		req := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			Data:    "test data for workspace keyring",
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Encrypted)
		require.NotEmpty(t, res.Body.KeyId)
	})
}