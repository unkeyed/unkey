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
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_delete_role"
)

func TestPermissionErrors(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB:        h.DB,
		Keys:      h.Keys,
		Auditlogs: h.Auditlogs,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a role for testing
	// Create a role to attempt to delete (in the same workspace)
	roleID := uid.New(uid.TestPrefix)
	roleName := "test.forbidden.role"
	roleDesc := "Test role for forbidden access"

	err := db.Query.InsertRole(ctx, h.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: workspace.ID,
		Name:        roleName,
		Description: sql.NullString{Valid: true, String: roleDesc},
	})
	require.NoError(t, err)

	t.Run("missing required permission", func(t *testing.T) {
		// Create a root key with a different permission (not rbac.*.delete_role)
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				Role: roleID,
			},
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})

	t.Run("no permissions", func(t *testing.T) {
		// Create a root key with no permissions
		rootKeyNoPerms := h.CreateRootKey(workspace.ID, "") // No permissions

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKeyNoPerms)},
		}

		res := testutil.CallRoute[handler.Request, openapi.ForbiddenErrorResponse](
			h,
			route,
			headers,
			handler.Request{
				Role: roleID,
			},
		)

		require.Equal(t, 403, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "permission")
	})
}
