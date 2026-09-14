package handler_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_permission"
)

func TestNotFoundErrors(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{
		DB: h.DB,
	}

	h.Register(route)

	// Create a workspace
	workspace := h.Resources().UserWorkspace

	// Create a root key with appropriate permissions
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_permission")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for non-existent permission ID
	t.Run("non-existent permission ID", func(t *testing.T) {
		req := handler.Request{
			Permission: "perm_does_not_exist",
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "does not exist")
	})

	t.Run("permission ID from another project", func(t *testing.T) {
		otherProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Other permission project",
			Slug:        uid.New("project"),
		})
		permissionID := uid.New(uid.PermissionPrefix)
		require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  workspace.ID,
			ProjectID:    otherProject.ID,
			Name:         "other.permission",
			Slug:         "other.permission",
			Description:  dbtype.NullString{Valid: false},
			CreatedAtM:   time.Now().UnixMilli(),
		}))

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{Permission: permissionID},
		)

		require.Equal(t, http.StatusNotFound, res.Status, res.RawBody)
	})

	// Test case for valid-looking but non-existent permission ID
	t.Run("valid-looking but non-existent permission ID", func(t *testing.T) {
		nonExistentID := uid.New(uid.PermissionPrefix) // Generate a valid ID format that doesn't exist

		req := handler.Request{
			Permission: nonExistentID,
		}

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			req,
		)

		require.Equal(t, 404, res.Status)
		require.NotNil(t, res.Body)
		require.NotNil(t, res.Body.Error)
		require.Contains(t, res.Body.Error.Detail, "does not exist")
	})
}
