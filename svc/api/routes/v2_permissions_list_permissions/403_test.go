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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_permissions"
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

	// Create some test permissions to later try to list
	err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: uid.New(uid.PermissionPrefix),
		WorkspaceID:  workspace.ID,
		Name:         "test.permission.auth",
		Slug:         "test-permission-auth",
		Description:  dbtype.NullString{Valid: true, String: "Test permission for authorization tests"},
		CreatedAtM:   time.Now().UnixMilli(),
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

		// Create permissions in the other workspace
		err := db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
			PermissionID: uid.New(uid.PermissionPrefix),
			WorkspaceID:  otherWorkspace.ID,
			Name:         "other.workspace.permission",
			Slug:         "other-workspace-permission",
			Description:  dbtype.NullString{Valid: true, String: "This permission is in a different workspace"},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		require.NoError(t, err)

		// Create a root key for the original workspace with read_permission
		rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		req := handler.Request{}

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
		for _, perm := range res.Body.Data {
			require.NotEqual(t, "other.workspace.permission", perm.Name)
		}
	})
}
