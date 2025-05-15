package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_create_role"
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
		Auditlogs:   h.Auditlogs,
	})

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

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
		require.Equal(t, res.Body.Error.Detail, "insufficient permissions")

		// Verify no role was created
		roles, err := h.DB.RO().QueryContext(ctx,
			`SELECT COUNT(*) FROM "roles" WHERE "name" = $1 AND "workspaceId" = $2`,
			req.Name, workspace.ID)
		require.NoError(t, err)
		require.True(t, roles.Next())

		var count int
		err = roles.Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "No role should have been created")
		roles.Close()
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace("other-workspace")

		// Create a root key for the other workspace with all permissions
		rootKey := h.CreateRootKey(otherWorkspace.ID, "rbac.*.create_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Name: "test.role.wrong.workspace",
		}

		// Make the request
		res, err := h.Client.Post(
			"/v2/permissions.createRole",
			"application/json",
			testutil.MustMarshal(req),
			headers,
		)

		require.NoError(t, err)

		// The role should be created in the authorized workspace (the other workspace)
		// not in the original workspace
		roles, err := h.DB.RO().QueryContext(ctx,
			`SELECT COUNT(*) FROM "roles" WHERE "name" = $1 AND "workspaceId" = $2`,
			req.Name, workspace.ID)
		require.NoError(t, err)
		require.True(t, roles.Next())

		var count int
		err = roles.Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "No role should have been created in original workspace")
		roles.Close()
	})
}
