package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
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

		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          permissionID,
			WorkspaceID: workspace.ID,
			Name:        permissionName,
			Description: db.NewNullString(permissionDesc),
		})
		require.NoError(t, err)

		// Verify the permission exists before deletion
		perm, err := db.Query.FindPermissionById(ctx, h.DB.RO(), permissionID)
		require.NoError(t, err)
		require.Equal(t, permissionID, perm.ID)

		// Now delete the permission
		req := handler.Request{
			PermissionId: permissionID,
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
		_, err = db.Query.FindPermissionById(ctx, h.DB.RO(), permissionID)
		require.Error(t, err, "Permission should no longer exist")
		require.True(t, db.IsNotFound(err), "Error should be 'not found'")

		// Verify audit log was created
		auditLogs, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "auditlogs" WHERE "event" = 'permission.delete' AND "resourceId" = $1`,
			permissionID)
		require.NoError(t, err)
		require.True(t, auditLogs.Next(), "Audit log for permission deletion should exist")
		auditLogs.Close()
	})

	// Test case for deleting a permission with relationships
	t.Run("delete permission with relationships", func(t *testing.T) {
		// Create a permission with role and key relationships
		permissionID := uid.New(uid.PermissionPrefix)
		permissionName := "test.delete.permission.with.relations"

		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          permissionID,
			WorkspaceID: workspace.ID,
			Name:        permissionName,
		})
		require.NoError(t, err)

		// Create a role
		roleID := "role_test"
		_, err = h.DB.RW().ExecContext(ctx,
			`INSERT INTO "roles" ("id", "workspaceId", "name") VALUES ($1, $2, $3)`,
			roleID, workspace.ID, "test-role")
		require.NoError(t, err)

		// Create a role-permission relationship
		_, err = h.DB.RW().ExecContext(ctx,
			`INSERT INTO "role_permissions" ("roleId", "permissionId") VALUES ($1, $2)`,
			roleID, permissionID)
		require.NoError(t, err)

		// Create a key-permission relationship
		keyID := "key_test"
		_, err = h.DB.RW().ExecContext(ctx,
			`INSERT INTO "key_permissions" ("keyId", "permissionId") VALUES ($1, $2)`,
			keyID, permissionID)
		require.NoError(t, err)

		// Now delete the permission
		req := handler.Request{
			PermissionId: permissionID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)

		// Verify the permission no longer exists
		_, err = db.Query.FindPermissionById(ctx, h.DB.RO(), permissionID)
		require.Error(t, err, "Permission should no longer exist")

		// Verify role-permission relationship no longer exists
		rolePerms, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "role_permissions" WHERE "permissionId" = $1`,
			permissionID)
		require.NoError(t, err)
		require.False(t, rolePerms.Next(), "Role-permission relationship should not exist")
		rolePerms.Close()

		// Verify key-permission relationship no longer exists
		keyPerms, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "key_permissions" WHERE "permissionId" = $1`,
			permissionID)
		require.NoError(t, err)
		require.False(t, keyPerms.Next(), "Key-permission relationship should not exist")
		keyPerms.Close()
	})
}
