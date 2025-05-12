package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_role"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
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

	// Create a test role to try to retrieve
	roleID := id.NewRole()
	roleName := "test.role.access"

	_, err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		ID:          roleID,
		WorkspaceID: workspace.ID,
		Name:        roleName,
		Description: db.NewNullString("Test role for authorization tests"),
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing read_role
	t.Run("missing read_role permission", func(t *testing.T) {
		// Create a root key with some permissions but not read_role
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			RoleId: roleID,
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

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspace.ID, "rbac.*.read_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			RoleId: roleID, // Role is in the original workspace
		}

		// When accessing from wrong workspace, the behavior could be a 404 Not Found
		// rather than 403 Forbidden, as the handler masks workspace mismatches as "not found"
		res, err := h.Client.Post(
			"/v2/permissions.getRole",
			"application/json",
			testutil.MustMarshal(req),
			headers,
		)

		require.NoError(t, err)
		require.Equal(t, 404, res.StatusCode, "Wrong workspace access should return 404")
	})
}
