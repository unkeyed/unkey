package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for getting a permission
	t.Run("get permission with all fields", func(t *testing.T) {
		// First, create a permission to retrieve
		permissionID := uid.New(uid.PermissionPrefix)
		permissionName := "test.get.permission"
		permissionDesc := "Test permission for get endpoint"
		createdAt := time.Now()

		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          permissionID,
			WorkspaceID: workspace.ID,
			Name:        permissionName,
			Description: db.NewNullString(permissionDesc),
			CreatedAtM:  db.NewNullTime(createdAt),
		})
		require.NoError(t, err)

		// Now retrieve the permission
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
		require.NotNil(t, res.Body.Data)
		require.NotNil(t, res.Body.Data.Permission)

		// Verify permission data
		permission := res.Body.Data.Permission
		require.Equal(t, permissionID, permission.Id)
		require.Equal(t, permissionName, permission.Name)
		require.Equal(t, workspace.ID, permission.WorkspaceId)
		require.Equal(t, permissionDesc, permission.Description)
		require.NotEmpty(t, permission.CreatedAt)
	})

	// Test case for getting a permission without description
	t.Run("get permission without description", func(t *testing.T) {
		// First, create a permission to retrieve, without a description
		permissionID := uid.New(uid.PermissionPrefix)
		permissionName := "test.get.permission.no.desc"

		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          permissionID,
			WorkspaceID: workspace.ID,
			Name:        permissionName,
			Description: db.NullString{}, // Empty description
		})
		require.NoError(t, err)

		// Now retrieve the permission
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
		require.NotNil(t, res.Body.Data)
		require.NotNil(t, res.Body.Data.Permission)

		// Verify permission data
		permission := res.Body.Data.Permission
		require.Equal(t, permissionID, permission.Id)
		require.Equal(t, permissionName, permission.Name)
		require.Equal(t, workspace.ID, permission.WorkspaceId)
		require.Empty(t, permission.Description)
	})
}
