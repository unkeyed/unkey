package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

func TestAuthorizationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Test case for insufficient permissions - missing create_role
	t.Run("missing create_role permission", func(t *testing.T) {
		// Create a root key with some permissions but not create_role
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.role.unauthorized",
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions")

		// Verify no role was created
		_, err := db.Query.FindRoleByNameAndWorkspaceID(context.Background(), h.DB.RO(), db.FindRoleByNameAndWorkspaceIDParams{
			Name:        req.Name,
			WorkspaceID: workspace.ID,
		})
		require.True(t, db.IsNotFound(err), "No role should have been created")
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace()

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspace.ID, "rbac.*.create_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.role.wrong.workspace",
		}

		// Make the request - this should succeed in the other workspace
		testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		// The role should be created in the authorized workspace (the other workspace)
		// not in the original workspace
		_, err := db.Query.FindRoleByNameAndWorkspaceID(context.Background(), h.DB.RO(), db.FindRoleByNameAndWorkspaceIDParams{
			Name:        req.Name,
			WorkspaceID: workspace.ID,
		})
		require.True(t, db.IsNotFound(err), "No role should have been created in original workspace")
	})
}
