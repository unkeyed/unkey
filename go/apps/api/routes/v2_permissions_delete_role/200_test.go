package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_role"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for deleting a role with permissions
	t.Run("delete role with permissions", func(t *testing.T) {
		// Create a role to delete
		roleID := id.NewRole()
		roleName := "test.delete.role"
		roleDesc := "Test role for deletion"
		createdAt := time.Now()

		_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			ID:          roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: db.NewNullString(roleDesc),
			CreatedAtM:  db.NewNullTime(createdAt),
		})
		require.NoError(t, err)

		// Create some permissions to assign to the role
		permIDs := []string{id.NewPermission(), id.NewPermission()}
		for i, permID := range permIDs {
			_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				ID:          permID,
				WorkspaceID: workspace.ID,
				Name:        fmt.Sprintf("test.perm.%d", i),
				Description: db.NewNullString(fmt.Sprintf("Test permission %d", i)),
				CreatedAtM:  db.NewNullTime(createdAt),
			})
			require.NoError(t, err)

			// Create role-permission relationship
			_, err = db.Query.InsertRolePermission(ctx, h.DB.RW(), db.InsertRolePermissionParams{
				RoleID:       roleID,
				PermissionID: permID,
			})
			require.NoError(t, err)
		}

		// Create a key with this role assigned
		keyID := id.NewKey()
		_, err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			WorkspaceID: workspace.ID,
			ApiID:       nil,
			Hash:        []byte("test_hash"),
			Prefix:      "test_prefix",
			Name:        db.NewNullString("test key"),
		})
		require.NoError(t, err)

		// Assign role to key
		_, err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:  keyID,
			RoleID: roleID,
		})
		require.NoError(t, err)

		// Verify role exists before deletion
		_, err = db.Query.FindRoleById(ctx, h.DB.RO(), roleID)
		require.NoError(t, err)

		// Delete the role
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
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify role has been deleted
		_, err = db.Query.FindRoleById(ctx, h.DB.RO(), roleID)
		require.Error(t, err)
		require.True(t, db.IsNotFound(err), "Role should not exist after deletion")

		// Verify role_permissions have been deleted
		for _, permID := range permIDs {
			rolePerms, err := db.Query.FindRolePermissionsByRoleIdAndPermissionId(ctx, h.DB.RO(), db.FindRolePermissionsByRoleIdAndPermissionIdParams{
				RoleID:       roleID,
				PermissionID: permID,
			})
			require.NoError(t, err)
			require.Empty(t, rolePerms, "Role-Permission relationships should be deleted")
		}

		// Verify key_roles have been deleted
		keyRoles, err := db.Query.FindKeyRolesByKeyIdAndRoleId(ctx, h.DB.RO(), db.FindKeyRolesByKeyIdAndRoleIdParams{
			KeyID:  keyID,
			RoleID: roleID,
		})
		require.NoError(t, err)
		require.Empty(t, keyRoles, "Key-Role relationships should be deleted")
	})

	// Test case for deleting a role without permissions or key assignments
	t.Run("delete role without relationships", func(t *testing.T) {
		// Create a role with no permissions or key assignments
		roleID := id.NewRole()
		roleName := "test.delete.role.no.rels"
		roleDesc := "Test role with no relationships for deletion"

		_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			ID:          roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: db.NewNullString(roleDesc),
		})
		require.NoError(t, err)

		// Verify role exists before deletion
		_, err = db.Query.FindRoleById(ctx, h.DB.RO(), roleID)
		require.NoError(t, err)

		// Delete the role
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
		require.NotEmpty(t, res.Body.Meta.RequestId)

		// Verify role has been deleted
		_, err = db.Query.FindRoleById(ctx, h.DB.RO(), roleID)
		require.Error(t, err)
		require.True(t, db.IsNotFound(err), "Role should not exist after deletion")
	})
}
