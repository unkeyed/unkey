package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	handler "github.com/unkeyed/unkey/go/apps/api/routes/v2_permissions_get_permission"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/testutil"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestPermissionErrors(t *testing.T) {
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

	// Create a test permission to try to retrieve
	permissionID := uid.New(uid.PermissionPrefix)
	permissionName := "test.permission.access"

	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		Name:         permissionName,
		Slug:         "test-permission-access",
		Description:  sql.NullString{Valid: true, String: "Test permission for authorization tests"},
		CreatedAtM:   time.Now().UnixMilli(),
	})
	require.NoError(t, err)

	// Test case for insufficient permissions - missing read_permission
	t.Run("missing required permission", func(t *testing.T) {
		// Create a root key with some permissions but not read_permission
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.create_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{
			PermissionId: permissionID,
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
		require.Contains(t, res.Body.Error.Detail, "permission")
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
			PermissionId: permissionID,
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
		require.Contains(t, res.Body.Error.Detail, "permission")
	})
}
