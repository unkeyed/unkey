package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_permission"
)

func TestAuthorizationErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Test case for insufficient permissions - missing create_permission
	t.Run("missing create_permission permission", func(t *testing.T) {
		// Create a root key with only read permissions but no create_permission permission
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.permission.unauthorized",
			Slug: "test-permission-unauthorized",
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusForbidden, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "Missing permission: 'rbac.*.create_permission'", res.Body.Error.Detail)
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Use a non-existent workspace ID
		otherWorkspaceID := "ws_nonexistent"

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspaceID, "rbac.*.create_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.permission.wrong.workspace",
			Slug: "test-permission-wrong-workspace",
		}

		// This is generally masked as a 404 or 403 depending on the implementation
		// Use CallRoute and check for error response
		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.True(t, res.Status == http.StatusForbidden || res.Status == http.StatusNotFound,
			"Expected 403 or 404 status code, got %d", res.Status)
	})
}
