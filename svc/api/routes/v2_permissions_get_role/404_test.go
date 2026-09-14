package handler_test

import (
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_role"
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
	rootKey := h.CreateRootKey(workspace.ID, "rbac.*.read_role")

	// Set up request headers
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	// Test case for non-existent role ID
	t.Run("non-existent role ID", func(t *testing.T) {
		req := handler.Request{
			Role: "role_does_not_exist",
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

	t.Run("role ID from another project", func(t *testing.T) {
		otherProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Other role project",
			Slug:        uid.New("project"),
		})
		roleID := uid.New(uid.RolePrefix)
		require.NoError(t, db.Query.InsertRole(t.Context(), h.DB.RW(), db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: workspace.ID,
			ProjectID:   otherProject.ID,
			Name:        "other-project-role",
			Description: sql.NullString{Valid: false},
			CreatedAt:   time.Now().UnixMilli(),
		}))

		res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](
			h,
			route,
			headers,
			handler.Request{Role: roleID},
		)

		require.Equal(t, http.StatusNotFound, res.Status, res.RawBody)
	})
}
