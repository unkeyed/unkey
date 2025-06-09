package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_role"
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
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for getting a role with permissions
	t.Run("get role with permissions", func(t *testing.T) {
		// First, create a role to retrieve
		roleID := uid.New(uid.TestPrefix)
		roleName := "test.get.role"
		roleDesc := "Test role for get endpoint"

		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: roleDesc},
		})
		require.NoError(t, err)

		// Create some permissions to assign to the role
		permIDs := []string{uid.New(uid.PermissionPrefix), uid.New(uid.PermissionPrefix)}
		for i, permID := range permIDs {
			err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				PermissionID: permID,
				WorkspaceID:  workspace.ID,
				Name:         fmt.Sprintf("test.perm.%d", i),
				Slug:         fmt.Sprintf("test-perm-%d", i),
				Description:  sql.NullString{Valid: true, String: fmt.Sprintf("Test permission %d", i)},
				CreatedAtM:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)

			// Create role-permission relationship
			err = db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{
				RoleID:       roleID,
				PermissionID: permID,
				WorkspaceID:  workspace.ID,
				CreatedAtM:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}

		// Now retrieve the role
		req := handler.Request{
			RoleId: roleID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotNil(t, res.Body.Data.Role)

		// Verify role data
		role := res.Body.Data.Role
		require.Equal(t, roleID, role.Id)
		require.Equal(t, roleName, role.Name)
		require.Equal(t, workspace.ID, role.WorkspaceId)
		require.NotNil(t, role.Description)
		require.Equal(t, roleDesc, *role.Description)
		require.NotEmpty(t, role.CreatedAt)

		// Verify permissions
		require.NotNil(t, role.Permissions)
		require.Len(t, role.Permissions, 2)

		// Create a map of permission IDs for easier checking
		permMap := make(map[string]bool)
		for _, perm := range role.Permissions {
			permMap[perm.Id] = true
			require.Equal(t, workspace.ID, perm.WorkspaceId)
		}

		// Verify all expected permissions are present
		for _, permID := range permIDs {
			require.True(t, permMap[permID], "Permission %s should be in the response", permID)
		}
	})

	// Test case for getting a role without permissions
	t.Run("get role without permissions", func(t *testing.T) {
		// Create a role with no permissions
		roleID := uid.New(uid.TestPrefix)
		roleName := "test.get.role.no.perms"
		roleDesc := "Test role with no permissions"

		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: roleDesc},
		})
		require.NoError(t, err)

		// Now retrieve the role
		req := handler.Request{
			RoleId: roleID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.NotNil(t, res.Body.Data.Role)

		// Verify role data
		role := res.Body.Data.Role
		require.Equal(t, roleID, role.Id)
		require.Equal(t, roleName, role.Name)
		require.Equal(t, workspace.ID, role.WorkspaceId)
		require.NotNil(t, role.Description)
		require.Equal(t, roleDesc, *role.Description)

		// Verify permissions array is empty
		require.NotNil(t, role.Permissions)
		require.Len(t, role.Permissions, 0)
	})
}
