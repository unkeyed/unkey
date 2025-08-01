package internalVaultEncrypt_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_encrypt"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestVaultEncryptBadRequest(t *testing.T) {
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

	t.Run("missing keyring", func(t *testing.T) {
		req := openapi.InternalVaultEncryptRequestBody{
			// Keyring: "", // Missing keyring
			Data: "test data",
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing data", func(t *testing.T) {
		req := openapi.InternalVaultEncryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			// Data: "", // Missing data (but empty string is valid, so this should succeed)
		}

		res := testutil.CallRoute[openapi.InternalVaultEncryptRequestBody, openapi.InternalVaultEncryptResponseBody](h, route, headers, req)
		require.Equal(t, 200, res.Status) // Empty data is valid
		require.NotNil(t, res.Body)
	})

	t.Run("invalid json body", func(t *testing.T) {
		// This test requires sending raw bytes instead of a structured request
		// We'll test by sending malformed JSON through the harness
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer test-token-123"},
		}

		// Send a request with invalid JSON structure
		invalidJson := map[string]interface{}{
			"keyring": 12345, // Should be string, not number
			"data":    true,  // Should be string, not boolean
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](h, route, headers, invalidJson)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}