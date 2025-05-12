package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_delete_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/id"
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

	// Test case for non-existent permission ID
	t.Run("non-existent permission ID", func(t *testing.T) {
		req := handler.Request{
			PermissionId: "perm_does_not_exist",
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

	// Test case for valid-looking but non-existent permission ID
	t.Run("valid-looking but non-existent permission ID", func(t *testing.T) {
		nonExistentID := id.NewPermission() // Generate a valid ID format that doesn't exist

		req := handler.Request{
			PermissionId: nonExistentID,
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

	// Test case for already deleted permission
	t.Run("already deleted permission", func(t *testing.T) {
		// First, create a permission
		permissionID := id.NewPermission()
		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          permissionID,
			WorkspaceID: workspace.ID,
			Name:        "test.permission.to.delete",
		})
		require.NoError(t, err)

		// Delete the permission
		_, err = db.Query.DeletePermission(ctx, h.DB.RW(), permissionID)
		require.NoError(t, err)

		// Try to delete it again
		req := handler.Request{
			PermissionId: permissionID,
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
