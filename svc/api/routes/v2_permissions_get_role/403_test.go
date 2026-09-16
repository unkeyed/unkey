package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_role"
)

// TestGetRoleMasksInsufficientPermissions guarantees that callers cannot use
// the response to find roles that they cannot read.
func TestGetRoleMasksInsufficientPermissions(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace
	projectID, err := projects.EnsureDefaultProject(ctx, h.DB.RW(), workspace.ID)
	require.NoError(t, err)

	// Create a test role to try to retrieve
	roleID := uid.New(uid.TestPrefix)
	roleName := "test.role.access"

	err = db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		ProjectID:   projectID,
		Name:        roleName,
		Description: sql.NullString{Valid: true, String: "Test role for authorization tests"},
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing required permission
	t.Run("missing required permission", func(t *testing.T) {
		// Create a root key with some permissions but not read_role
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			Role: roleID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/role_not_found", res.Body.Error.Type)
		require.Equal(t, "The requested role does not exist.", res.Body.Error.Detail)
	})

	// Test case for no permissions
	t.Run("no permissions", func(t *testing.T) {
		// Create a root key with no permissions
		rootKeyNoPerms := h.CreateRootKey(workspace.ID, "") // No permissions

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyNoPerms)},
		}

		req := handler.Request{
			Role: roleID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, http.StatusNotFound, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Equal(t, "https://unkey.com/docs/errors/unkey/data/role_not_found", res.Body.Error.Type)
		require.Equal(t, "The requested role does not exist.", res.Body.Error.Detail)
	})
}
