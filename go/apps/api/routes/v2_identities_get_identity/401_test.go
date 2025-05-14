package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_get_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

// Helper function for creating string pointers
func strPtr(s string) *string {
	return &s
}

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger,
		DB:          h.DB,
		Keys:        h.Keys,
		Permissions: h.Permissions,
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		req := handler.Request{
			IdentityId: strPtr("identity_123"),
		}

		// Call without auth header
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, nil, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/missing_credential", res.Body.Error.Type)
		require.Equal(t, "You need to provide credentials to access this resource.", res.Body.Error.Detail)
	})

	t.Run("malformed Authorization header", func(t *testing.T) {
		req := handler.Request{
			IdentityId: strPtr("identity_123"),
		}

		// Invalid format
		headers := http.Header{
			"Authorization": {"InvalidFormat xyz"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/bearer_format", res.Body.Error.Type)
	})

	t.Run("invalid root key", func(t *testing.T) {
		req := handler.Request{
			IdentityId: strPtr("identity_123"),
		}

		// Non-existent key
		headers := http.Header{
			"Authorization": {"Bearer invalid_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authentication/invalid_key", res.Body.Error.Type)
	})
}
