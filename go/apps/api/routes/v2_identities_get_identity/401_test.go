package handler_test

import (
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
	route := &handler.Handler{
		Logger: h.Logger,
		DB:     h.DB,
		Keys:   h.Keys,
	}

	h.Register(route)

	t.Run("invalid root key", func(t *testing.T) {
		req := handler.Request{
			IdentityId: strPtr("identity_123"),
		}

		// Non-existent key
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/authentication/key_not_found", res.Body.Error.Type)
	})
}
