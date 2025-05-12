package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/apps/api/routes/v2_identities_update_identity/handler"
	"github.com/unkeyed/unkey/go/internal/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestNotFound(t *testing.T) {
	h := testutil.NewHarness(t)
	route := handler.New(handler.Services{
		Logger:      h.Logger(),
		DB:          h.Database(),
		Keys:        h.Keys(),
		Permissions: h.Permissions(),
		Auditlogs:   h.Auditlogs(),
	})

	rootKeyID := h.CreateRootKey()
	headers := testutil.RootKeyAuth(rootKeyID)

	// Setup permissions to allow updating any identity
	h.SetupPermissions(t, rootKeyID, h.DefaultWorkspaceID(), "identity.*.update_identity", true)

	t.Run("identity ID does not exist", func(t *testing.T) {
		nonExistentID := uid.New(uid.IdentityPrefix)
		req := handler.Request{
			identityID: &nonExistentID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, got: %d", res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity/not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "not found")
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})

	t.Run("external ID does not exist", func(t *testing.T) {
		nonExistentExternalID := "non_existent_external_id"
		req := handler.Request{
			externalID: &nonExistentExternalID,
			meta: map[string]interface{}{
				"test": "value",
			},
		}
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, got: %d", res.Status)
		require.Equal(t, "https://unkey.com/docs/api-reference/errors-v2/unkey/data/identity/not_found", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "not found")
		require.Equal(t, http.StatusNotFound, res.Body.Error.Status)
		require.Equal(t, "Not Found", res.Body.Error.Title)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	})
}
