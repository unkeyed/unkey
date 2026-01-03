package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	handler "github.com/unkeyed/unkey/apps/api/routes/v2_permissions_list_permissions"
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

	// Create test permissions
	testPermissions := []struct {
		ID          string
		Name        string
		Description string
	}{
		{uid.New(uid.PermissionPrefix), "test.permission.1", "Description for test permission 1"},
		{uid.New(uid.PermissionPrefix), "test.permission.2", "Description for test permission 2"},
		{uid.New(uid.PermissionPrefix), "test.permission.3", "Description for test permission 3"},
		{uid.New(uid.PermissionPrefix), "test.permission.4", "Description for test permission 4"},
		{uid.New(uid.PermissionPrefix), "test.permission.5", "Description for test permission 5"},
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

	// Create permissions in a different workspace to test isolation
	otherWorkspace := h.CreateWorkspace()
	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: uid.New(uid.PermissionPrefix),
		WorkspaceID:  otherWorkspace.ID,
		Name:         "other.workspace.permission",
		Slug:         "other-workspace-permission",
		Description:  dbtype.NullString{Valid: true, String: "This permission is in a different workspace"},
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Test case for listing all permissions
	t.Run("list all permissions", func(t *testing.T) {
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
		require.Len(t, res.Body.Data, len(testPermissions))
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore) // No more pages
		require.Nil(t, res.Body.Pagination.Cursor)

		// Verify we got the correct permissions
		permMap := make(map[string]bool)
		for _, perm := range res.Body.Data {
			permMap[perm.Id] = true
		}

		// Check that all created permissions are in the response
		for _, perm := range testPermissions {
			require.True(t, permMap[perm.ID], "Permission %s not found in response", perm.ID)
		}
	})

	// Test case for empty results in a new workspace
	t.Run("empty results", func(t *testing.T) {
		emptyWorkspace := h.CreateWorkspace()
		emptyKey := h.CreateRootKey(emptyWorkspace.ID, "rbac.*.read_permission")

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
		// Create 101 additional permissions to test pagination
		for i := 0; i < 101; i++ {
			permID := uid.New(uid.PermissionPrefix)
			err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				PermissionID: permID,
				WorkspaceID:  workspace.ID,
				Name:         fmt.Sprintf("bulk.permission.%d", i),
				Slug:         fmt.Sprintf("bulk-permission-%d", i),
				Description:  dbtype.NullString{Valid: true, String: fmt.Sprintf("Bulk permission %d", i)},
				CreatedAtM:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}

		// First page - should return 100 permissions with cursor
		req1 := handler.Request{}

		res1 := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req1,
		)

		require.Equal(t, 200, res1.Status)
		require.NotNil(t, res1.Body.Pagination.Cursor)
		require.Len(t, res1.Body.Data, 100)
		require.True(t, res1.Body.Pagination.HasMore)

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
		require.Greater(t, len(res2.Body.Data), 0) // Should have some permissions

		// Verify first and second page have different permissions
		for _, perm1 := range res1.Body.Data {
			for _, perm2 := range res2.Body.Data {
				require.NotEqual(t, perm1.Id, perm2.Id, "Permission should not appear on both pages")
			}
		}
	})
}
