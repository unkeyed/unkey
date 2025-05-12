package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_permissions"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Create test permissions
	testPermissions := []struct {
		ID          string
		Name        string
		Description string
	}{
		{id.NewPermission(), "test.permission.1", "Description for test permission 1"},
		{id.NewPermission(), "test.permission.2", "Description for test permission 2"},
		{id.NewPermission(), "test.permission.3", "Description for test permission 3"},
		{id.NewPermission(), "test.permission.4", "Description for test permission 4"},
		{id.NewPermission(), "test.permission.5", "Description for test permission 5"},
	}

	// Insert test permissions into the database
	for _, perm := range testPermissions {
		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          perm.ID,
			WorkspaceID: workspace.ID,
			Name:        perm.Name,
			Description: db.NewNullString(perm.Description),
			CreatedAtM:  db.NewNullTime(time.Now()),
		})
		require.NoError(t, err)
	}

	// Create permissions in a different workspace to test isolation
	otherWorkspace := h.CreateWorkspace("other-workspace")
	_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		ID:          id.NewPermission(),
		WorkspaceID: otherWorkspace.ID,
		Name:        "other.workspace.permission",
		Description: db.NewNullString("This permission is in a different workspace"),
	})
	require.NoError(t, err)

	// Test case for listing all permissions
	t.Run("list all permissions", func(t *testing.T) {
		req := handler.Request{
			Limit: 100,
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
		require.Len(t, res.Body.Data.Permissions, len(testPermissions))
		require.Equal(t, len(testPermissions), res.Body.Data.Total)
		require.Nil(t, res.Body.Data.Cursor) // No more pages

		// Verify we got the correct permissions
		permMap := make(map[string]bool)
		for _, perm := range res.Body.Data.Permissions {
			permMap[perm.Id] = true
			require.Equal(t, workspace.ID, perm.WorkspaceId)
		}

		// Check that all created permissions are in the response
		for _, perm := range testPermissions {
			require.True(t, permMap[perm.ID], "Permission %s not found in response", perm.ID)
		}
	})

	// Test case for limiting results
	t.Run("limit results", func(t *testing.T) {
		req := handler.Request{
			Limit: 2,
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
		require.Len(t, res.Body.Data.Permissions, 2)
		require.Equal(t, len(testPermissions), res.Body.Data.Total)
		require.NotNil(t, res.Body.Data.Cursor)
	})

	// Test case for pagination with cursor
	t.Run("pagination with cursor", func(t *testing.T) {
		// First page
		req1 := handler.Request{
			Limit: 2,
		}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status)
		require.NotNil(t, res1.Body.Data.Cursor)
		require.Len(t, res1.Body.Data.Permissions, 2)

		// Second page
		req2 := handler.Request{
			Limit:  2,
			Cursor: res1.Body.Data.Cursor,
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
		require.Len(t, res2.Body.Data.Permissions, 2)
		require.NotNil(t, res2.Body.Data.Cursor)

		// Verify first and second page have different permissions
		for _, perm1 := range res1.Body.Data.Permissions {
			for _, perm2 := range res2.Body.Data.Permissions {
				require.NotEqual(t, perm1.Id, perm2.Id, "Permission should not appear on both pages")
			}
		}

		// Third page (should have 1 remaining permission)
		req3 := handler.Request{
			Limit:  2,
			Cursor: res2.Body.Data.Cursor,
		}

		res3 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req3,
		)

		require.Equal(t, 200, res3.Status)
		require.NotNil(t, res3.Body)
		require.NotNil(t, res3.Body.Data)
		require.Len(t, res3.Body.Data.Permissions, 1)
		require.Nil(t, res3.Body.Data.Cursor) // No more pages
	})

	// Test empty results in a new workspace
	t.Run("empty results", func(t *testing.T) {
		emptyWorkspace := h.CreateWorkspace("empty-workspace")
		emptyKey := h.CreateRootKey(emptyWorkspace.ID, "rbac.*.read_permission")

		emptyHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", emptyKey)},
		}

		req := handler.Request{
			Limit: 100,
		}

		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			emptyHeaders,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)
		require.Len(t, res.Body.Data.Permissions, 0)
		require.Equal(t, 0, res.Body.Data.Total)
		require.Nil(t, res.Body.Data.Cursor)
	})
}
