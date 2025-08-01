package internalVaultDecrypt_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/_internal_vault_decrypt"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestVaultDecryptBadRequest(t *testing.T) {
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
		req := openapi.InternalVaultDecryptRequestBody{
			// Keyring: "", // Missing keyring
			Encrypted: "fake-encrypted-data",
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("missing encrypted data", func(t *testing.T) {
		req := openapi.InternalVaultDecryptRequestBody{
			Keyring: h.Resources().UserWorkspace.ID,
			// Encrypted: "", // Missing encrypted data
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("invalid json body", func(t *testing.T) {
		// Test with invalid JSON structure
		invalidJson := map[string]interface{}{
			"keyring":   12345,     // Should be string, not number
			"encrypted": []string{}, // Should be string, not array
		}

		res := testutil.CallRoute[map[string]interface{}, openapi.BadRequestErrorResponse](h, route, headers, invalidJson)
		require.Equal(t, 400, res.Status)
		require.NotNil(t, res.Body)
	})
}

func TestVaultDecryptInvalidEncryptedData(t *testing.T) {
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

	t.Run("invalid base64 encrypted data", func(t *testing.T) {
		req := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: "not-valid-base64-data!@#$%",
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("valid base64 but invalid encrypted format", func(t *testing.T) {
		req := openapi.InternalVaultDecryptRequestBody{
			Keyring:   h.Resources().UserWorkspace.ID,
			Encrypted: "dGhpcyBpcyBub3QgdmFsaWQgZW5jcnlwdGVkIGRhdGE=", // Base64 of "this is not valid encrypted data"
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})

	t.Run("wrong keyring for encrypted data", func(t *testing.T) {
		// This test would require actual encrypted data from a different keyring
		// For now, we'll use fake data that should fail during vault processing
		req := openapi.InternalVaultDecryptRequestBody{
			Keyring:   "wrong-keyring",
			Encrypted: "dGhpcyBpcyBub3QgdmFsaWQgZW5jcnlwdGVkIGRhdGE=",
		}

		res := testutil.CallRoute[openapi.InternalVaultDecryptRequestBody, openapi.InternalServerErrorResponse](h, route, headers, req)
		require.Equal(t, 500, res.Status)
		require.NotNil(t, res.Body)
	})
}