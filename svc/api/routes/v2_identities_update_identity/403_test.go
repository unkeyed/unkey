package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_identities_update_identity"
)

func TestForbidden(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	t.Run("no permission to update identity", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
		})

		// Create root key without permissions
		rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID)
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("wrong permission type", func(t *testing.T) {
		externalID := uid.New(uid.TestPrefix)
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: h.Resources().UserWorkspace.ID,
			ExternalID:  externalID,
		})

		// Create root key with wrong permission
		rootKeyID := h.CreateRootKey(h.Resources().UserWorkspace.ID, "identity.*.create_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusForbidden, res.Status)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/authorization/insufficient_permissions", res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("with permission to update identity", func(t *testing.T) {
		workspaceID := h.Resources().UserWorkspace.ID
		externalID := "test_user_403"
		h.CreateIdentity(seed.CreateIdentityRequest{
			WorkspaceID: workspaceID,
			ExternalID:  externalID,
		})

		// Create root key with correct permission
		rootKeyID := h.CreateRootKey(workspaceID, "identity.*.update_identity")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyID)},
		}

		meta := map[string]interface{}{
			"test": "value",
		}
		req := handler.Request{
			Identity: externalID,
			Meta:     &meta,
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, http.StatusOK, res.Status, "expected 200, got: %d, response: %s", res.Status, res.RawBody)
		require.Equal(t, externalID, res.Body.Data.ExternalId)
	})
}
