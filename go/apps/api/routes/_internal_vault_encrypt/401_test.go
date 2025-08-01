package internalVaultEncrypt_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_encrypt"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestVaultEncryptUnauthorized(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Vault:  h.Vault,
		Logger: h.Logger,
		Token:  "correct-token-123",
	}

	h.Register(route)

	req := openapi.InternalVaultEncryptRequestBody{
		Keyring: h.Resources().UserWorkspace.ID,
		Data:    "test data",
	}

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer wrong-token"},
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"correct-token-123"}, // Missing "Bearer " prefix
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer "},
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("case sensitive token comparison", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer Correct-Token-123"}, // Wrong case
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("bearer with extra spaces", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"  Bearer   correct-token-123  "}, // Should still work
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
	})
}