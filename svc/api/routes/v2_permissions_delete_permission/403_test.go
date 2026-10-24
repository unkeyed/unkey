package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

// TestDeletePermissionMasksInsufficientPermissions guarantees that a legacy
// read does not reveal a permission, while a caller who can read it by URN
// learns they lack delete. A caller with no access gets not-found.
func TestDeletePermissionMasksInsufficientPermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace
	projectID, err := projects.EnsureDefaultProject(ctx, h.DB.RW(), workspace.ID)
	require.NoError(t, err)

	// Create a test permission to try to delete
	permissionID := uid.New(uid.PermissionPrefix)
	permissionName := "test.permission.delete.auth"

	err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		ProjectID:    projectID,
		Name:         permissionName,
		Slug:         "test-permission-delete-auth",
		Description:  dbtype.NullString{Valid: true, String: "Test permission for authorization tests"},
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	t.Run("legacy read does not reveal the permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{Permission: permissionID},
		)

		require.Equal(t, http.StatusNotFound, res.Status)
		require.Equal(t, "The requested permission does not exist.", res.Body.Error.Detail)
		require.NotContains(t, res.RawBody, permissionID)

		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
	})

	t.Run("urn read names the missing permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/%s#read", workspace.ID, projectID, permissionID))

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{Permission: permissionID},
		)

		require.Equal(t, http.StatusForbidden, res.Status)
		require.Contains(t, res.Body.Error.Detail, "#delete")
		require.Contains(t, res.Body.Error.Detail, "Log in again")
		require.NotContains(t, res.RawBody, permissionID)

		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
	})

	t.Run("no access is not found", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID)

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{Permission: permissionID},
		)

		require.Equal(t, http.StatusNotFound, res.Status)
		require.Equal(t, "The requested permission does not exist.", res.Body.Error.Detail)
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
