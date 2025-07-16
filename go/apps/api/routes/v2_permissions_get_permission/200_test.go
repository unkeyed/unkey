package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for getting a permission
	t.Run("get permission with all fields", func(t *testing.T) {
		// First, create a permission to retrieve
		permissionNameSlug := "test.get.permission"
		permissionDesc := "Test permission for get endpoint"

		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: uid.New(uid.PermissionPrefix),
			WorkspaceID:  workspace.ID,
			Name:         permissionNameSlug,
			Slug:         permissionNameSlug,
			Description:  sql.NullString{Valid: true, String: permissionDesc},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Now retrieve the permission
		req := handler.Request{
			Slug: permissionNameSlug,
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
		require.Equal(t, permissionNameSlug, permission.Slug)
		require.Equal(t, permissionNameSlug, permission.Name)
		require.NotNil(t, permission.Description)
		require.Equal(t, permissionDesc, *permission.Description)
		require.NotNil(t, permission.CreatedAt)
	})

	// Test case for getting a permission without description
	t.Run("get permission without description", func(t *testing.T) {
		// First, create a permission to retrieve, without a description
		permissionNameSlug := "test.get.permission.no.desc"

		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: uid.New(uid.PermissionPrefix),
			WorkspaceID:  workspace.ID,
			Name:         permissionNameSlug,
			Slug:         "test-get-permission-no-desc",
			Description:  sql.NullString{}, // Empty description
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Now retrieve the permission
		req := handler.Request{
			Slug: permissionNameSlug,
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
		require.Equal(t, permissionNameSlug, permission.Slug)
		require.Equal(t, permissionNameSlug, permission.Name)
		require.Nil(t, permission.Description)
	})
}
