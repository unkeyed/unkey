package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_permissions"

	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

func TestSuccess(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB: h.DB,
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

	t.Run("search", func(t *testing.T) {
		// Fresh workspace so search results are not polluted by other subtests
		searchWorkspace := h.CreateWorkspace()
		searchKey := h.CreateRootKey(searchWorkspace.ID, "rbac.*.read_permission")
		searchHeaders := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", searchKey)},
		}

		searchPermissions := []struct {
			Name        string
			Slug        string
			Description string
		}{
			{"documents.read", "documents-read", "Read stored documents"},
			{"billing.manage", "billing-manage", "Change plans and invoices"},
			{"reports_100%", "reports-full", "Export everything"},
		}
		permissionIDs := make(map[string]string)
		for _, perm := range searchPermissions {
			permissionID := uid.New(uid.PermissionPrefix)
			permissionIDs[perm.Name] = permissionID
			err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
				PermissionID: permissionID,
				WorkspaceID:  searchWorkspace.ID,
				Name:         perm.Name,
				Slug:         perm.Slug,
				Description:  dbtype.NullString{Valid: true, String: perm.Description},
				CreatedAtM:   time.Now().UnixMilli(),
			})
			require.NoError(t, err)
		}

		list := func(t *testing.T, search string) []string {
			t.Helper()
			req := handler.Request{Search: &search}
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, searchHeaders, req)
			require.Equal(t, 200, res.Status, "expected 200, got: %d", res.Status)
			names := make([]string, 0, len(res.Body.Data))
			for _, perm := range res.Body.Data {
				names = append(names, perm.Name)
			}
			return names
		}

		t.Run("matches name substring", func(t *testing.T) {
			require.Equal(t, []string{"documents.read"}, list(t, "documents."))
		})

		t.Run("matches permission id", func(t *testing.T) {
			require.Equal(t, []string{"billing.manage"}, list(t, permissionIDs["billing.manage"]))
		})

		t.Run("matches slug substring", func(t *testing.T) {
			require.Equal(t, []string{"reports_100%"}, list(t, "reports-full"))
		})

		t.Run("matches description substring", func(t *testing.T) {
			require.Equal(t, []string{"billing.manage"}, list(t, "invoices"))
		})

		t.Run("is case insensitive", func(t *testing.T) {
			require.Equal(t, []string{"billing.manage"}, list(t, "BILLING"))
		})

		t.Run("wildcards match literally", func(t *testing.T) {
			// Unescaped, "%" would match every permission and "d_cuments" would match "documents"
			require.Equal(t, []string{"reports_100%"}, list(t, "100%"))
			require.Empty(t, list(t, "d_cuments"))
		})

		t.Run("no matches returns empty list", func(t *testing.T) {
			require.Empty(t, list(t, "does-not-exist"))
		})

		t.Run("whitespace-only search returns all", func(t *testing.T) {
			require.ElementsMatch(t,
				[]string{"documents.read", "billing.manage", "reports_100%"},
				list(t, "   "),
			)
		})
	})
}
