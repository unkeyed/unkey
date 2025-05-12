package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_roles"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
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

	// Create test roles and permissions
	roleIDs := []string{}
	permissionIDs := []string{}
	createdAt := time.Now()

	// Create permissions
	for i := 0; i < 5; i++ {
		permID := id.NewPermission()
		permissionIDs = append(permissionIDs, permID)

		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          permID,
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("test.perm.%d", i),
			Description: db.NewNullString(fmt.Sprintf("Test permission %d", i)),
			CreatedAtM:  db.NewNullTime(createdAt),
		})
		require.NoError(t, err)
	}

	// Create roles with permissions
	for i := 0; i < 10; i++ {
		roleID := id.NewRole()
		roleIDs = append(roleIDs, roleID)

		_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			ID:          roleID,
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("test.role.%d", i),
			Description: db.NewNullString(fmt.Sprintf("Test role %d", i)),
			CreatedAtM:  db.NewNullTime(createdAt),
		})
		require.NoError(t, err)

		// Assign 2 permissions to each role
		for j := 0; j < 2; j++ {
			permIdx := (i + j) % len(permissionIDs)
			_, err = db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{
				RoleID:       roleID,
				PermissionID: permissionIDs[permIdx],
			})
			require.NoError(t, err)
		}
	}

	// Test case for listing roles without pagination
	t.Run("list roles without pagination", func(t *testing.T) {
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
		require.NotNil(t, res.Body.Data.Roles)

		// Default limit should be 10
		require.LessOrEqual(t, len(res.Body.Data.Roles), 10)
		require.Equal(t, 10, len(res.Body.Data.Roles))

		// Total should be the total count of roles
		require.Equal(t, 10, res.Body.Data.Total)

		// Verify role data
		for _, role := range res.Body.Data.Roles {
			require.NotEmpty(t, role.Id)
			require.NotEmpty(t, role.Name)
			require.Equal(t, workspace.ID, role.WorkspaceId)
			require.NotNil(t, role.Description)
			require.NotNil(t, role.CreatedAt)

			// Verify permissions
			require.NotNil(t, role.Permissions)
			require.Len(t, role.Permissions, 2)

			for _, perm := range role.Permissions {
				require.NotEmpty(t, perm.Id)
				require.NotEmpty(t, perm.Name)
				require.Equal(t, workspace.ID, perm.WorkspaceId)
				require.NotNil(t, perm.Description)
			}
		}
	})

	// Test case for pagination with limit
	t.Run("list roles with limit", func(t *testing.T) {
		limit := int32(5)
		req := handler.Request{
			Limit: &limit,
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
		require.NotNil(t, res.Body.Data.Roles)

		// Should respect the limit
		require.Equal(t, int(limit), len(res.Body.Data.Roles))

		// Should have cursor for next page
		require.NotNil(t, res.Body.Data.Cursor)
		require.NotEmpty(t, *res.Body.Data.Cursor)
	})

	// Test case for pagination with cursor
	t.Run("list roles with cursor", func(t *testing.T) {
		// First page
		limit := int32(3)
		req1 := handler.Request{
			Limit: &limit,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status)
		require.NotNil(t, res1.Body.Data.Cursor)

		// Second page using cursor from first page
		req2 := handler.Request{
			Limit:  &limit,
			Cursor: res1.Body.Data.Cursor,
		}

		res2 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req2,
		)

		require.Equal(t, 200, res2.Status)
		require.NotNil(t, res2.Body.Data)
		require.NotNil(t, res2.Body.Data.Roles)
		require.Equal(t, int(limit), len(res2.Body.Data.Roles))

		// Ensure first and second pages have different roles
		firstPageRoleIDs := make(map[string]bool)
		for _, role := range res1.Body.Data.Roles {
			firstPageRoleIDs[role.Id] = true
		}

		for _, role := range res2.Body.Data.Roles {
			require.False(t, firstPageRoleIDs[role.Id], "Role from second page should not be in first page")
		}
	})

	// Test with empty roles
	t.Run("list roles in empty workspace", func(t *testing.T) {
		// Create a new workspace with no roles
		emptyWorkspace := h.CreateWorkspace()
		emptyRootKey := h.CreateRootKey(emptyWorkspace.ID, "rbac.*.read_role")

		emptyHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", emptyRootKey)},
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
		require.NotNil(t, res.Body.Data.Roles)
		require.Empty(t, res.Body.Data.Roles)
		require.Equal(t, 0, res.Body.Data.Total)
		require.Nil(t, res.Body.Data.Cursor)
	})
}
