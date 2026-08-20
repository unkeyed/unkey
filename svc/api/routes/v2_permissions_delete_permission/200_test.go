package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_permission"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
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

		unrelatedPermissionID := uid.New(uid.PermissionPrefix)
		err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: unrelatedPermissionID,
			WorkspaceID:  workspace.ID,
			Name:         "unrelated.permission",
			Slug:         "unrelated-permission",
			Description:  dbtype.NullString{},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		roleID := uid.New(uid.TestPrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        "permission deletion test role",
			Description: sql.NullString{},
		})
		require.NoError(t, err)

		keyID := uid.New(uid.KeyPrefix)
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeySpaceID:  workspace.ID,
			Hash:        hash.Sha256(uid.New(uid.TestPrefix)),
			Start:       "test_",
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "permission deletion test key"},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     true,
		})
		require.NoError(t, err)

		for _, relationshipPermissionID := range []string{permissionID, unrelatedPermissionID} {
			err = db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{
				RoleID:       roleID,
				PermissionID: relationshipPermissionID,
				WorkspaceID:  workspace.ID,
				CreatedAtM:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)

			err = db.Query.InsertKeyPermission(ctx, h.DB.RW(), db.InsertKeyPermissionParams{
				KeyID:        keyID,
				PermissionID: relationshipPermissionID,
				WorkspaceID:  workspace.ID,
				CreatedAt:    time.Now().UnixMilli(),
				UpdatedAt:    sql.NullInt64{},
			})
			require.NoError(t, err)
		}

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

		rolePermissions, err := db.Query.FindRolePermissionByRoleAndPermissionID(ctx, h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
			RoleID:       roleID,
			PermissionID: permissionID,
		})
		require.NoError(t, err)
		require.Empty(t, rolePermissions, "role-permission relationship should be deleted")

		rolePermissions, err = db.Query.FindRolePermissionByRoleAndPermissionID(ctx, h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
			RoleID:       roleID,
			PermissionID: unrelatedPermissionID,
		})
		require.NoError(t, err)
		require.Len(t, rolePermissions, 1, "unrelated role-permission relationship should remain")

		keyPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err)
		require.Len(t, keyPermissions, 1)
		require.Equal(t, unrelatedPermissionID, keyPermissions[0].ID, "unrelated key-permission relationship should remain")

		_, err = db.Query.FindRoleByID(ctx, h.DB.RO(), roleID)
		require.NoError(t, err, "related role should remain")
		_, err = db.Query.FindKeyByID(ctx, h.DB.RO(), keyID)
		require.NoError(t, err, "related key should remain")

		// Verify the audit log was queued in clickhouse_outbox.
		auditLogs := h.FindAuditLogsByTargetID(ctx, t, permissionID)
		require.NotEmpty(t, auditLogs, "Audit log for permission deletion should exist")
		foundDeleteEvent := false
		for _, ev := range auditLogs {
			if ev.Event == "permission.delete" {
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
