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
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for deleting a permission
	t.Run("delete permission", func(t *testing.T) {
		// First, create a permission to delete
		permissionID := uid.New(uid.PermissionPrefix)
		permissionName := "test.delete.permission"
		permissionDesc := "Test permission to be deleted"

		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         permissionName,
			Slug:         "test-delete-permission",
			Description:  dbtype.NullString{Valid: true, String: permissionDesc},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify the permission exists before deletion
		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)

		// Now delete the permission
		req := handler.Request{
			Permission: permissionID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify the permission no longer exists
		_, err = db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.Error(t, err, "Permission should no longer exist")
		require.True(t, db.IsNotFound(err), "Error should be 'not found'")

		// Verify audit log was created
		auditLogs, err := db.Query.FindAuditLogTargetByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.NotEmpty(t, auditLogs, "Audit log for permission deletion should exist")

		foundDeleteEvent := false
		for _, log := range auditLogs {
			if log.AuditLog.Event == "permission.delete" {
				foundDeleteEvent = true
				break
			}
		}
		require.True(t, foundDeleteEvent, "Should find a permission.delete audit log event")
	})

	// Test case for deleting a permission with description
	t.Run("delete permission with description", func(t *testing.T) {
		// Create a permission with a description
		permissionID := uid.New(uid.PermissionPrefix)
		permissionName := "test.delete.permission.with.description"
		permissionDesc := "This permission has a description"

		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         permissionName,
			Slug:         "test-delete-permission-with-description",
			Description:  dbtype.NullString{Valid: true, String: permissionDesc},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify the permission exists before deletion
		perm, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)
		require.Equal(t, permissionDesc, perm.Description.String)

		// Delete the permission
		req := handler.Request{
			Permission: permissionID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)

		// Verify the permission no longer exists
		_, err = db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
		require.Error(t, err, "Permission should no longer exist")
		require.True(t, db.IsNotFound(err), "Error should be 'not found'")
	})
}
