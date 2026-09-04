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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_create_role"
)

// TestPermissionSlugFromAnotherProjectCreatesLocalPermission guarantees role
// creation does not attach a permission owned by another project.
func TestPermissionSlugFromAnotherProjectCreatesLocalPermission(t *testing.T) {
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
	permissionSlug := "other.project.create.role.permission"
	otherPermissionID := uid.New(uid.PermissionPrefix)
	require.NoError(t, db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
		PermissionID: otherPermissionID,
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
	permissions := []string{permissionSlug}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Name:        "role-with-other-project-permission",
		Permissions: &permissions,
	})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	assigned, err := db.Query.ListDirectPermissionsByRoleID(ctx, h.DB.RO(), res.Body.Data.RoleId)
	require.NoError(t, err)
	require.Len(t, assigned, 1)
	require.NotEqual(t, otherPermissionID, assigned[0].ID)
	assignedPermission, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), assigned[0].ID)
	require.NoError(t, err)
	require.NotEqual(t, otherProject.ID, assignedPermission.ProjectID)
}
