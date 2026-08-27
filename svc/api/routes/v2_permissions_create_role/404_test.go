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
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

func TestPermissionFromAnotherProjectIsNotAttached(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)
	workspace := h.Resources().UserWorkspace
	otherProject := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Other Create Role Project",
		Slug:        "other-create-role-project",
	})
	permissionID := uid.New(uid.PermissionPrefix)
	permissionSlug := "other.project.create.role.permission"
	require.NoError(t, db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspace.ID,
		ProjectID:    otherProject.ID,
		Name:         permissionSlug,
		Slug:         permissionSlug,
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	rootKey := h.CreateRootKey(
		workspace.ID,
		"rbac.*.create_role",
		"rbac.*.add_permission_to_role",
		"rbac.*.create_permission",
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}
	roleName := "role-with-other-project-permission"
	permissions := []string{permissionSlug}

	res := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{
		Name:        roleName,
		Permissions: &permissions,
	})

	require.Equal(t, http.StatusNotFound, res.Status, res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/data/permission_not_found", res.Body.Error.Type)
	_, err := db.Query.FindRoleByNameAndWorkspaceID(ctx, h.DB.RO(), db.FindRoleByNameAndWorkspaceIDParams{
		Name:        roleName,
		WorkspaceID: workspace.ID,
	})
	require.True(t, db.IsNotFound(err))
}
