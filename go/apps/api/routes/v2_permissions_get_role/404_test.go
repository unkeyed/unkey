package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_role"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestNotFoundErrors(t *testing.T) {
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

	// Test case for non-existent role ID
	t.Run("non-existent role ID", func(t *testing.T) {
		req := handler.Request{
			RoleId: "role_does_not_exist",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "not found")
	})

	// Test case for valid-looking but non-existent role ID
	t.Run("valid-looking but non-existent role ID", func(t *testing.T) {
		nonExistentID := id.NewRole() // Generate a valid ID format that doesn't exist

		req := handler.Request{
			RoleId: nonExistentID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "not found")
	})

	// Test case for already deleted role
	t.Run("already deleted role", func(t *testing.T) {
		// First, create a role
		roleID := id.NewRole()
		_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			ID:          roleID,
			WorkspaceID: workspace.ID,
			Name:        "test.role.to.delete",
		})
		require.NoError(t, err)

		// Delete the role
		_, err = h.DB.RW().ExecContext(ctx,
			`DELETE FROM "roles" WHERE "id" = $1`,
			roleID)
		require.NoError(t, err)

		// Try to retrieve the deleted role
		req := handler.Request{
			RoleId: roleID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, res.Body.Error.Detail, "not found")
	})
}
