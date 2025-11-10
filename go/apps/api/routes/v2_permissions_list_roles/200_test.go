package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:     h.DB,
		Keys:   h.Keys,
		Logger: h.Logger,
	}

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

	// Create test permissions first
	testPermissions := []struct {
		ID          string
		Name        string
		Description string
	}{
		{uid.New(uid.PermissionPrefix), "test.permission.1", "Description for test permission 1"},
		{uid.New(uid.PermissionPrefix), "test.permission.2", "Description for test permission 2"},
		{uid.New(uid.PermissionPrefix), "test.permission.3", "Description for test permission 3"},
	}

	// Insert test permissions into the database
	for i, perm := range testPermissions {
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: perm.ID,
			WorkspaceID:  workspace.ID,
			Name:         perm.Name,
			Slug:         fmt.Sprintf("test-permission-%d", i+1),
			Description:  dbtype.NullString{Valid: true, String: perm.Description},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)
	}

	// Create test roles
	testRoles := []struct {
		ID          string
		Name        string
		Description string
	}{
		{uid.New(uid.TestPrefix), "test.role.1", "Description for test role 1"},
		{uid.New(uid.TestPrefix), "test.role.2", "Description for test role 2"},
		{uid.New(uid.TestPrefix), "test.role.3", "Description for test role 3"},
		{uid.New(uid.TestPrefix), "test.role.4", "Description for test role 4"},
		{uid.New(uid.TestPrefix), "test.role.5", "Description for test role 5"},
	}

	// Insert test roles into the database
	for _, role := range testRoles {
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      role.ID,
			WorkspaceID: workspace.ID,
			Name:        role.Name,
			Description: sql.NullString{Valid: true, String: role.Description},
		})
		require.NoError(t, err)

		// Assign 2 permissions to each role
		for i := 0; i < 2 && i < len(testPermissions); i++ {
			err = db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{
				RoleID:       role.ID,
				PermissionID: testPermissions[i].ID,
				WorkspaceID:  workspace.ID,
				CreatedAtM:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}
	}

	// Create roles in a different workspace to test isolation
	otherWorkspace := h.CreateWorkspace()
	err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		RoleID:      uid.New(uid.TestPrefix),
		WorkspaceID: otherWorkspace.ID,
		Name:        "other.workspace.role",
		Description: sql.NullString{Valid: true, String: "This role is in a different workspace"},
	})
	require.NoError(t, err)

	// Test case for listing all roles
	t.Run("list all roles", func(t *testing.T) {
		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, len(testRoles))

		// Verify we got the correct roles
		roleMap := make(map[string]bool)
		for _, role := range res.Body.Data {
			roleMap[role.Id] = true
			require.NotNil(t, role.Description)

			// Verify permissions are attached
			require.NotNil(t, role.Permissions)
			require.Len(t, role.Permissions, 2)

			for _, perm := range role.Permissions {
				require.NotEmpty(t, perm.Id)
				require.NotEmpty(t, perm.Name)
				require.NotNil(t, perm.Description)
			}
		}

		// Check that all created roles are in the response
		for _, role := range testRoles {
			require.True(t, roleMap[role.ID], "Role %s not found in response", role.ID)
		}
	})

	// Test case for empty results in a new workspace
	t.Run("empty results", func(t *testing.T) {
		emptyWorkspace := h.CreateWorkspace()
		emptyKey := h.CreateRootKey(emptyWorkspace.ID, "rbac.*.read_role")

		emptyHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", emptyKey)},
		}

		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			emptyHeaders,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data, 0)
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	// Test case for pagination with cursor
	t.Run("pagination with cursor", func(t *testing.T) {
		// Create 101 additional roles to test pagination
		for i := 0; i < 101; i++ {
			roleID := uid.New(uid.TestPrefix)
			err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
				RoleID:      roleID,
				WorkspaceID: workspace.ID,
				Name:        fmt.Sprintf("bulk.role.%d", i),
				Description: sql.NullString{Valid: true, String: fmt.Sprintf("Bulk role %d", i)},
			})
			require.NoError(t, err)

			// Add one permission to each bulk role
			if len(testPermissions) > 0 {
				err = db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{
					RoleID:       roleID,
					PermissionID: testPermissions[0].ID,
					WorkspaceID:  workspace.ID,
					CreatedAtM:   time.Now().UnixMilli(),
				})
				require.NoError(t, err)
			}
		}

		// First page - should return 100 roles with cursor
		req1 := handler.Request{}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status)
		require.NotNil(t, res1.Body.Pagination)
		require.NotNil(t, res1.Body.Pagination.Cursor)
		require.True(t, res1.Body.Pagination.HasMore)
		require.Len(t, res1.Body.Data, 100)

		// Second page
		req2 := handler.Request{
			Cursor: res1.Body.Pagination.Cursor,
		}

		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req2,
		)

		require.Equal(t, 200, res2.Status)
		require.NotNil(t, res2.Body)
		require.NotNil(t, res2.Body.Data)
		require.Greater(t, len(res2.Body.Data), 0) // Should have some roles

		// Verify first and second page have different roles
		for _, role1 := range res1.Body.Data {
			for _, role2 := range res2.Body.Data {
				require.NotEqual(t, role1.Id, role2.Id, "Role should not appear on both pages")
			}
		}
	})
}
