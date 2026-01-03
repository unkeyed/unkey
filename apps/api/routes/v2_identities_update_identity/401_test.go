package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/api/openapi"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_identities_update_identity"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestUnauthorized(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		Logger:    h.Logger,
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	t.Run("missing Authorization header", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}

		// Call without auth header
		headers := http.Header{
			"Content-Type": {"application/json"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Authorization header")
	})

	t.Run("malformed Authorization header", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}

		// Invalid format
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"InvalidFormat xyz"},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/malformed", res.Body.Error.Type)
	})

	t.Run("invalid root key", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}

		// Non-existent key
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer invalid_key"},
		}
		res := testutil.CallRoute[handler.Request, openapi.UnauthorizedErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusUnauthorized, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/key_not_found", res.Body.Error.Type)
	})

	t.Run("empty bearer token", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer "},
		}
		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authentication/malformed", res.Body.Error.Type)
	})

	t.Run("key from different workspace", func(t *testing.T) {
		// Create a different workspace
		differentWorkspace := h.CreateWorkspace()

		// Create a root key for different workspace
		differentWorkspaceKey := h.CreateRootKey(differentWorkspace.ID, "identity.*.update_identity")

		externalID := uid.New(uid.TestPrefix)
		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", differentWorkspaceKey)},
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/identity_not_found", res.Body.Error.Type)
	})
}
