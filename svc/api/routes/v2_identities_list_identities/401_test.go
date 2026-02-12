package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_list_identities"
)

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:   h.DB,
		Keys: h.Keys,
	}

	// Register the route with the harness
	h.Register(route)

	t.Run("missing Authorization header", func(t *testing.T) {
		req := handler.Request{}

		// Call without auth header
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, nil, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		// The specific error type may vary, so we just check it's a valid error response
		require.NotEmpty(t, res.Body.Error.Type)
		require.NotEmpty(t, res.Body.Error.Detail)
	})

	t.Run("malformed Authorization header", func(t *testing.T) {
		req := handler.Request{}

		// Invalid format
		headers := http.Header{
			"Authorization": []string{"InvalidFormat xyz"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		// The specific error type may vary, so we just check it's a valid error response
		require.NotEmpty(t, res.Body.Error.Type)
	})

	t.Run("invalid root key", func(t *testing.T) {
		req := handler.Request{}

		// Non-existent key
		headers := http.Header{
			"Authorization": []string{"Bearer invalid_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		// Invalid key returns 401 (Unauthorized)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		// The specific error type may vary, so we just check it's a valid error response
		require.NotEmpty(t, res.Body.Error.Type)
	})
}
