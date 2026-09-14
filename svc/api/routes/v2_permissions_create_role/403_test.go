package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

func TestAuthorizationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace
	assertRoleDoesNotExist := func(t *testing.T, workspaceID string, name string) {
		t.Helper()
		var exists bool
		err := h.DB.RO().QueryRowContext(
			t.Context(),
			"SELECT EXISTS(SELECT 1 FROM roles WHERE workspace_id = ? AND name = ?)",
			workspaceID,
			name,
		).Scan(&exists)
		require.NoError(t, err)
		require.False(t, exists, "No role should have been created")
	}

	// Test case for insufficient permissions - missing create_role
	t.Run("missing create_role permission", func(t *testing.T) {
		// Create a root key with some permissions but not create_role
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.role.unauthorized",
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "Missing one of these permissions")

		assertRoleDoesNotExist(t, workspace.ID, req.Name)
	})

	t.Run("missing add_permission_to_role permission", func(t *testing.T) {
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role", "rbac.*.create_permission")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		permissions := []string{"documents.read.missing.add"}
		req := handler.Request{Name: "test.role.missing.add", Permissions: &permissions}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusForbidden, res.Status)
		assertRoleDoesNotExist(t, workspace.ID, req.Name)
	})

	t.Run("missing create_permission permission", func(t *testing.T) {
		existingPermission := h.CreatePermission(seed.CreatePermissionRequest{
			WorkspaceID: workspace.ID,
			Name:        "documents.read.existing.create.role",
			Slug:        "documents.read.existing.create.role",
		})
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role", "rbac.*.add_permission_to_role")
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		permissions := []string{existingPermission.Slug, "documents.write.missing.create.role"}
		req := handler.Request{Name: "test.role.missing.create", Permissions: &permissions}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](h, route, headers, req)

		require.Equal(t, http.StatusForbidden, res.Status)
		assertRoleDoesNotExist(t, workspace.ID, req.Name)
		missing, err := db.Query.FindPermissionsBySlugs(context.Background(), h.DB.RO(), db.FindPermissionsBySlugsParams{
			WorkspaceID: workspace.ID,
			ProjectID:   existingPermission.ProjectID,
			Slugs:       []string{"documents.write.missing.create.role"},
		})
		require.NoError(t, err)
		require.Empty(t, missing)
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace()

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspace.ID, "rbac.*.create_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.role.wrong.workspace",
		}

		// Make the request - this should succeed in the other workspace
		testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		// The role should be created in the authorized workspace (the other workspace)
		// not in the original workspace
		assertRoleDoesNotExist(t, workspace.ID, req.Name)
	})
}
