package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/testutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_roles"
)

func TestAuthorizationErrors(t *testing.T) {
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

	// Create some test roles to later try to list
	err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		RoleID:      uid.New(uid.TestPrefix),
		WorkspaceID: workspace.ID,
		Name:        "test.role.auth",
		Description: sql.NullString{Valid: true, String: "Test role for authorization tests"},
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing read_role
	t.Run("missing read_role permission", func(t *testing.T) {
		// Create a root key with some permissions but not read_role
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role") // Only has create, not read

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	// Test case for wrong workspace
	t.Run("wrong workspace", func(t *testing.T) {
		// Create a different workspace
		otherWorkspace := h.CreateWorkspace()

		// Create roles in the other workspace
		err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
			RoleID:      uid.New(uid.TestPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "other.workspace.role",
			Description: sql.NullString{Valid: true, String: "This role is in a different workspace"},
		})
		require.NoError(t, err)

		// Create a root key for the original workspace with read_role
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{}

		// When listing roles, we should only see roles from the authorized workspace
		// This should return 200 OK with only roles from the authorized workspace
		res := testutil.CallRoute[handler.Request, handler.Response](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 200, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Data)

		// Verify we only see roles from our workspace
		for _, role := range res.Body.Data {
			require.NotEqual(t, "other.workspace.role", role.Name)
		}
	})
}
