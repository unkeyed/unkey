package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := handler.New(handler.Services{
		DB:          h.DB,
		Keys:        h.Keys,
		Logger:      h.Logger,
		Permissions: h.Permissions,
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a test permission to try to delete
	permissionID := uid.New(uid.PermissionPrefix)
	permissionName := "test.permission.delete.auth"

	_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		ID:          permissionID,
		WorkspaceID: workspace.ID,
		Name:        permissionName,
		Description: db.NewNullString("Test permission for authorization tests"),
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing delete_permission
	t.Run("missing delete_permission permission", func(t *testing.T) {
		// Create a root key with some permissions but not delete_permission
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			PermissionId: permissionID,
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
		require.Equal(t, res.Body.Error.Detail, "insufficient permissions")

		// Verify the permission still exists (wasn't deleted)
		perm, err := db.Query.FindPermissionById(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace("other-workspace")

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspace.ID, "rbac.*.delete_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			PermissionId: permissionID, // Permission is in the original workspace
		}

		// When accessing from wrong workspace, the behavior could be a 404 Not Found
		// rather than 403 Forbidden, as the handler masks workspace mismatches as "not found"
		res, err := h.Client.Post(
			"/v2/permissions.deletePermission",
			"application/json",
			testutil.MustMarshal(req),
			headers,
		)

		require.NoError(t, err)
		require.Equal(t, 404, res.StatusCode, "Wrong workspace access should return 404")

		// Verify the permission still exists (wasn't deleted)
		perm, err := db.Query.FindPermissionById(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
	})
}
