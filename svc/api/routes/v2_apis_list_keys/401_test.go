package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_list_keys"
)

func TestAuthenticationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		Logger:   h.Logger,
		DB:       h.DB,
		Keys:     h.Keys,
		Vault:    h.Vault,
		ApiCache: h.Caches.LiveApiByID,
	}

	h.Register(route)

	// Create a valid request
	req := handler.Request{
		ApiId: "api_1234",
	}

	// Test case for invalid authorization token
	t.Run("invalid authorization token", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_token_that_does_not_exist"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		if res.Status == 401 {
			require.NotNil(t, res.Body.Error)
			require.Contains(t, res.Body.Error.Detail, "key")
		}
	})

	// Test case for expired or invalid key format
	t.Run("invalid key", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer not_a_valid_key"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "key")
	})

	// Test case for key with valid format but doesn't exist
	t.Run("valid format non-existent key", func(t *testing.T) {
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer sk_test_1234567890abcdef1234567890abcdef"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "key")
	})

	// Test case for verifying error response structure
	t.Run("verify error response structure", func(t *testing.T) {
		// Use a clearly invalid token to ensure we get 401
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer clearly_invalid_token_format"},
		}

		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 401, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.NotEmpty(t, res.Body.Error.Detail)
		require.Equal(t, 401, res.Body.Error.Status)
		require.NotEmpty(t, res.Body.Error.Title)

		// Verify meta information is included
		require.NotNil(t, res.Body.Meta)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
