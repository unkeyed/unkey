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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_role"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.delete_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for deleting a role with permissions
	t.Run("delete role with permissions", func(t *testing.T) {
		// Create a role to delete
		roleID := uid.New(uid.TestPrefix)
		roleName := "test.delete.role"
		roleDesc := "Test role for deletion"

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
			err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				PermissionID: permID,
				WorkspaceID:  workspace.ID,
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

		// Create a key with this role assigned
		keyID := uid.New(uid.KeyPrefix)
		err = db.Query.InsertKey(ctx, h.DB.RW(), db.InsertKeyParams{
			ID:          keyID,
			KeySpaceID:  workspace.ID, // Using workspace ID as keyring ID for test
			Hash:        hash.Sha256(uid.New(uid.TestPrefix)),
			Start:       "test_",
			WorkspaceID: workspace.ID,
			Name:        sql.NullString{Valid: true, String: "test key"},
			CreatedAtM:  time.Now().UnixMilli(),
			Enabled:     true,
		})
		require.NoError(t, err)

		// Assign role to key
		err = db.Query.InsertKeyRole(ctx, h.DB.RW(), db.InsertKeyRoleParams{
			KeyID:       keyID,
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			CreatedAtM:  time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Verify role exists before deletion
		_, err = db.Query.FindRoleByID(ctx, h.DB.RO(), roleID)
		require.NoError(t, err)

		// Delete the role
		req := handler.Request{
			Role: roleID,
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
		_, err = db.Query.FindRoleByID(ctx, h.DB.RO(), roleID)
		require.Error(t, err)
		require.True(t, db.IsNotFound(err), "Role should not exist after deletion")

		// Verify role_permissions have been deleted
		for _, permID := range permIDs {
			rolePerms, findErr := db.Query.FindRolePermissionByRoleAndPermissionID(ctx, h.DB.RO(), db.FindRolePermissionByRoleAndPermissionIDParams{
				RoleID:       roleID,
				PermissionID: permID,
			})
			require.NoError(t, findErr)
			require.Empty(t, rolePerms, "Role-Permission relationships should be deleted")
		}

		// Verify key_roles have been deleted
		keyRoles, err := db.Query.FindKeyRoleByKeyAndRoleID(ctx, h.DB.RO(), db.FindKeyRoleByKeyAndRoleIDParams{
			KeyID:  keyID,
			RoleID: roleID,
		})
		require.NoError(t, err)
		require.Empty(t, keyRoles, "Key-Role relationships should be deleted")
	})

	// Test case for deleting a role without permissions or key assignments
	t.Run("delete role without relationships", func(t *testing.T) {
		// Create a role with no permissions or key assignments
		roleID := uid.New(uid.TestPrefix)
		roleName := "test.delete.role.no.rels"
		roleDesc := "Test role with no relationships for deletion"

		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			Name:        roleName,
			Description: sql.NullString{Valid: true, String: roleDesc},
		})
		require.NoError(t, err)

		// Verify role exists before deletion
		_, err = db.Query.FindRoleByID(ctx, h.DB.RO(), roleID)
		require.NoError(t, err)

		// Delete the role
		req := handler.Request{
			Role: roleID,
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
		_, err = db.Query.FindRoleByID(ctx, h.DB.RO(), roleID)
		require.Error(t, err)
		require.True(t, db.IsNotFound(err), "Role should not exist after deletion")
	})
}
