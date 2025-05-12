package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_list_permissions"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
)

func TestAuthorizationErrors(t *testing.T) {
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

	// Create some test permissions to later try to list
	_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		ID:          id.NewPermission(),
		WorkspaceID: workspace.ID,
		Name:        "test.permission.auth",
		Description: db.NewNullString("Test permission for authorization tests"),
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing read_permission
	t.Run("missing read_permission permission", func(t *testing.T) {
		// Create a root key with some permissions but not read_permission
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_permission") // Only has create, not read

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Limit: 10,
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
		require.Equal(t, res.Body.Error.Detail, "insufficient permissions")
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace("other-workspace")

		// Create permissions in the other workspace
		_, err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			ID:          id.NewPermission(),
			WorkspaceID: otherWorkspace.ID,
			Name:        "other.workspace.permission",
			Description: db.NewNullString("This permission is in a different workspace"),
		})
		require.NoError(t, err)

		// Create a root key for the original workspace with read_permission
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Limit: 10,
		}

		// When listing permissions, we should only see permissions from the authorized workspace
		// This should return 200 OK with only permissions from the authorized workspace
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)

		// Verify we only see permissions from our workspace
		for _, perm := range res.Body.Data.Permissions {
			require.Equal(t, workspace.ID, perm.WorkspaceId)
			require.NotEqual(t, "other.workspace.permission", perm.Name)
		}
	})
}
