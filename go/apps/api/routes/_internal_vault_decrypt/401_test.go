package internalVaultDecrypt_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_decrypt"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestVaultDecryptUnauthorized(t *testing.T) {
	t.Parallel()

	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Vault:  h.Vault,
		Logger: h.Logger,
		Token:  "correct-token-123",
	}

	h.Register(route)

	req := openapi.InternalVaultDecryptRequestBody{
		Keyring:   h.Resources().UserWorkspace.ID,
		Encrypted: "fake-encrypted-data",
	}

	t.Run("missing authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type": {"application/json"},
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("invalid bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer wrong-token"},
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"correct-token-123"}, // Missing "Bearer " prefix
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("empty bearer token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer "},
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("case sensitive token comparison", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer Correct-Token-123"}, // Wrong case
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("bearer with extra spaces", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"  Bearer   correct-token-123  "}, // Should still work
		}

		// This should work since zen.StaticAuth handles whitespace properly
		// But we need valid encrypted data to avoid vault errors
		validReq := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: "fake-but-valid-format", // This will still fail at vault level, but auth should pass
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, validReq)
		// Should pass auth (not 401) but fail at vault level (500)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})
}