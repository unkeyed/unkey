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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_role"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace
	projectID, err := projects.EnsureDefaultProject(ctx, h.DB.RW(), workspace.ID)
	require.NoError(t, err)

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
			ProjectID:   projectID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: roleDesc},
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create some permissions to assign to the role
		permIDs := []string{uid.New(uid.PermissionPrefix), uid.New(uid.PermissionPrefix)}
		for i, permID := range permIDs {
			err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				PermissionID: permID,
				WorkspaceID:  workspace.ID,
				ProjectID:    projectID,
				Name:         fmt.Sprintf("test.perm.%d", i),
				Slug:         fmt.Sprintf("test-perm-%d", i),
				Description:  dbtype.NullString{Valid: true, String: fmt.Sprintf("Test permission %d", i)},
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
			Role: roleID,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		t.Logf("res: %s", res.RawBody)
		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)

		// Verify role data
		role := res.Body.Data
		require.Equal(t, roleID, role.Id)
		require.Equal(t, roleName, role.Name)
		require.NotNil(t, role.Description)
		require.Equal(t, roleDesc, role.Description)

		// Verify permissions
		require.NotNil(t, role.Permissions)
		require.Len(t, role.Permissions, 2)

		// Create a map of permission IDs for easier checking
		permMap := make(map[string]bool)
		for _, perm := range role.Permissions {
			permMap[perm.Id] = true
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
		roleName := "rolewithoutpermissions"
		roleDesc := "Test role with no permissions"

		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			ProjectID:   projectID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: roleDesc},
		})
		require.NoError(t, err)

		// Now retrieve the role
		req := handler.Request{
			Role: roleName,
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

		// Verify role data
		role := res.Body.Data
		require.Equal(t, roleID, role.Id)
		require.Equal(t, roleName, role.Name)
		require.NotNil(t, role.Description)
		require.Equal(t, roleDesc, role.Description)

		// Verify permissions array is empty
		require.Nil(t, role.Permissions)
	})

	t.Run("resolve duplicate name in default project", func(t *testing.T) {
		otherProjectID := uid.New(uid.ProjectPrefix)
		h.CreateProject(seed.CreateProjectRequest{
			ID:          otherProjectID,
			WorkspaceID: workspace.ID,
			Name:        "Other",
			Slug:        uid.New("other"),
		})

		name := "shared-role-name"
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      uid.New(uid.RolePrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   otherProjectID,
			Name:        name,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		defaultRoleID := uid.New(uid.RolePrefix)
		err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      defaultRoleID,
			WorkspaceID: workspace.ID,
			ProjectID:   projectID,
			Name:        name,
			CreatedAt:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			handler.Request{Role: name},
		)

		require.Equal(t, http.StatusOK, res.Status, res.RawBody)
		require.Equal(t, defaultRoleID, res.Body.Data.Id)
	})
}
