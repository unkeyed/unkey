package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestAuthorizationErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Logger:    h.Logger,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a test permission to try to delete
	permissionID := uid.New(uid.PermissionPrefix)
	permissionName := "test.permission.delete.auth"

	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         permissionName,
		Slug:         "test-permission-delete-auth",
		Description:  dbtype.NullString{Valid: true, String: "Test permission for authorization tests"},
		CreatedAtM:   time.Now().UnixMilli(),
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
			Permission: permissionID,
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

		// Verify the permission still exists (wasn't deleted)
		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace()

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspace.ID, "rbac.*.delete_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Permission: permissionID, // Permission is in the original workspace
		}

		// When accessing from wrong workspace, the behavior should be a 404 Not Found
		// as the handler masks workspace mismatches as "not found"
		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status, "Wrong workspace access should return 404")
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "does not exist")

		// Verify the permission still exists (wasn't deleted)
		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
	})
}
