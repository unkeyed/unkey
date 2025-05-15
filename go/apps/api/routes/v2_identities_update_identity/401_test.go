package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
		Auditlogs:   h.Auditlogs(),
	})

	t.Run("missing Authorization header", func(t *testing.T) {
		identityID := "identity_123"
		req := handler.Request{
			identityID: &identityID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}

		// Call without auth header
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, nil, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/missing_credential", res.Body.Error.Type)
		require.Equal(t, "You need to provide credentials to access this resource.", res.Body.Error.Detail)
	})

	t.Run("malformed Authorization header", func(t *testing.T) {
		identityID := "identity_123"
		req := handler.Request{
			identityID: &identityID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}

		// Invalid format
		headers := map[string]string{
			"Authorization": "InvalidFormat xyz",
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authorization/bearer_format", res.Body.Error.Type)
	})

	t.Run("invalid root key", func(t *testing.T) {
		identityID := "identity_123"
		req := handler.Request{
			identityID: &identityID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}

		// Non-existent key
		headers := map[string]string{
			"Authorization": "Bearer invalid_key",
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/auth/authentication/invalid_key", res.Body.Error.Type)
	})
}
