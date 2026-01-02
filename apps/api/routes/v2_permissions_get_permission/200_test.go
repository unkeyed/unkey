package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_permissions_get_permission"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// First, create a permission to retrieve
	permissionID := uid.New(uid.PermissionPrefix)
	permissionName := "test.get.permission"
	permissionDesc := "Test permission for get endpoint"
	permissionSlug := "test-get-permission"

	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         permissionName,
		Slug:         permissionSlug,
		Description:  dbtype.NullString{Valid: true, String: permissionDesc},
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Test case for getting a permission
	t.Run("get permission with all fields by ID", func(t *testing.T) {

		// Now retrieve the permission
		req := handler.Request{
			Permission: permissionID,
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

		// Verify permission data
		permission := res.Body.Data
		require.Equal(t, permissionID, permission.Id)
		require.Equal(t, permissionName, permission.Name)
		require.NotNil(t, permission.Description)
		require.Equal(t, permissionDesc, permission.Description)
	})

	t.Run("get permission with all fields by slug", func(t *testing.T) {
		req := handler.Request{Permission: permissionSlug}
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)

		// Verify permission data
		permission := res.Body.Data
		require.Equal(t, permissionID, permission.Id)
		require.Equal(t, permissionName, permission.Name)
		require.NotNil(t, permission.Description)
		require.Equal(t, permissionDesc, permission.Description)
	})

	// Test case for getting a permission without description
	t.Run("get permission without description", func(t *testing.T) {
		// First, create a permission to retrieve, without a description
		permissionID := uid.New(uid.PermissionPrefix)
		permissionName := "test.get.permission.no.desc"

		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			Name:         permissionName,
			Slug:         "test-get-permission-no-desc",
			Description:  dbtype.NullString{}, // Empty description
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Now retrieve the permission
		req := handler.Request{
			Permission: permissionID,
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

		// Verify permission data
		permission := res.Body.Data
		require.Equal(t, permissionID, permission.Id)
		require.Equal(t, permissionName, permission.Name)
		require.Empty(t, permission.Description)
	})
}
