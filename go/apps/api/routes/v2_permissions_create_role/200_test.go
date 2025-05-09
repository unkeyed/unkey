package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_role"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/id"
	"github.com/unkeyed/unkey/go/pkg/testutil"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for creating a role without permissions
	t.Run("create role without permissions", func(t *testing.T) {
		roleName := "test.role.no.permissions"
		description := "Test role without permissions"
		req := handler.Request{
			Name:        roleName,
			Description: &description,
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
		require.NotEmpty(t, res.Body.Data.RoleId)
		require.True(t, len(res.Body.Data.RoleId) > 0, "RoleId should not be empty")

		// Verify role was created in database
		role, err := h.DB.RO().QueryContext(ctx,
			`SELECT "id", "name", "description", "workspaceId" FROM "roles" WHERE "id" = $1`,
			res.Body.Data.RoleId)
		require.NoError(t, err)
		require.True(t, role.Next(), "Role should exist in database")

		var id, name, desc, wsID string
		err = role.Scan(&id, &name, &desc, &wsID)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, id)
		require.Equal(t, req.Name, name)
		require.Equal(t, description, desc)
		require.Equal(t, workspace.ID, wsID)
		role.Close()

		// Verify no role-permission relationships exist
		rolePerms, err := h.DB.RO().QueryContext(ctx,
			`SELECT COUNT(*) FROM "role_permissions" WHERE "roleId" = $1`,
			res.Body.Data.RoleId)
		require.NoError(t, err)
		require.True(t, rolePerms.Next(), "Should get a count result")

		var count int
		err = rolePerms.Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "Should have no role-permission relationships")
		rolePerms.Close()

		// Verify audit log was created
		auditLogs, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "auditlogs" WHERE "event" = 'role.create' AND "resourceId" = $1`,
			res.Body.Data.RoleId)
		require.NoError(t, err)
		require.True(t, auditLogs.Next(), "Audit log for role creation should exist")
		auditLogs.Close()
	})

	// Test case for creating a role with permissions
	t.Run("create role with permissions", func(t *testing.T) {
		roleName := "test.role.with.permissions"
		description := "Test role with permissions"

		// First, create some permissions to assign to the role
		permissionIDs := []string{
			id.NewPermission(),
			id.NewPermission(),
		}

		// Insert the permissions
		for i, permID := range permissionIDs {
			_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				ID:          permID,
				WorkspaceID: workspace.ID,
				Name:        fmt.Sprintf("test.perm.%d", i),
				Description: db.NewNullString(fmt.Sprintf("Test permission %d", i)),
			})
			require.NoError(t, err)
		}

		// Create the role with permissions
		req := handler.Request{
			Name:          roleName,
			Description:   &description,
			PermissionIds: &permissionIDs,
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
		require.NotEmpty(t, res.Body.Data.RoleId)

		// Verify role was created in database
		role, err := h.DB.RO().QueryContext(ctx,
			`SELECT "id", "name", "description", "workspaceId" FROM "roles" WHERE "id" = $1`,
			res.Body.Data.RoleId)
		require.NoError(t, err)
		require.True(t, role.Next(), "Role should exist in database")

		var id, name, desc, wsID string
		err = role.Scan(&id, &name, &desc, &wsID)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, id)
		require.Equal(t, req.Name, name)
		require.Equal(t, description, desc)
		require.Equal(t, workspace.ID, wsID)
		role.Close()

		// Verify role-permission relationships exist
		for _, permID := range permissionIDs {
			rolePerms, err := h.DB.RO().QueryContext(ctx,
				`SELECT * FROM "role_permissions" WHERE "roleId" = $1 AND "permissionId" = $2`,
				res.Body.Data.RoleId, permID)
			require.NoError(t, err)
			require.True(t, rolePerms.Next(), "Role-permission relationship should exist")
			rolePerms.Close()
		}

		// Verify audit log was created
		auditLogs, err := h.DB.RO().QueryContext(ctx,
			`SELECT * FROM "auditlogs" WHERE "event" = 'role.create' AND "resourceId" = $1`,
			res.Body.Data.RoleId)
		require.NoError(t, err)
		require.True(t, auditLogs.Next(), "Audit log for role creation should exist")
		auditLogs.Close()
	})

	// Test case for creating a role without description
	t.Run("create role without description", func(t *testing.T) {
		req := handler.Request{
			Name: "test.role.no.desc",
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
		require.NotEmpty(t, res.Body.Data.RoleId)

		// Verify role was created in database
		role, err := h.DB.RO().QueryContext(ctx,
			`SELECT "id", "name", "description", "workspaceId" FROM "roles" WHERE "id" = $1`,
			res.Body.Data.RoleId)
		require.NoError(t, err)
		require.True(t, role.Next(), "Role should exist in database")

		var id, name, desc, wsID string
		err = role.Scan(&id, &name, &desc, &wsID)
		require.NoError(t, err)
		require.Equal(t, res.Body.Data.RoleId, id)
		require.Equal(t, req.Name, name)
		require.Empty(t, desc, "Description should be empty")
		require.Equal(t, workspace.ID, wsID)
		role.Close()
	})
}
