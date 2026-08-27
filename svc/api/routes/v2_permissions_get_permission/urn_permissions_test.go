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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_permission"
)

// TestGetPermissionAuthorizesCanonicalReadPermission guarantees an exact
// project-scoped permission grant can read its permission.
func TestGetPermissionAuthorizesCanonicalReadPermission(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	permission := h.CreatePermission(seed.CreatePermissionRequest{
		WorkspaceID: workspaceID,
		Name:        "canonical.permission",
		Slug:        "canonical.permission",
	})
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/%s#read_permission", workspaceID, permission.ProjectID, permission.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Permission: permission.ID})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, permission.ID, res.Body.Data.Id)
}

// TestGetPermissionUsesActualProject guarantees a permission grant can read a
// permission in its actual project instead of the workspace's default project.
func TestGetPermissionUsesActualProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	projectID := uid.New(uid.ProjectPrefix)
	h.CreateProject(seed.CreateProjectRequest{
		ID:          projectID,
		WorkspaceID: workspaceID,
		Name:        "Other",
		Slug:        uid.New("other"),
	})
	permissionID := uid.New(uid.PermissionPrefix)
	require.NoError(t, db.Query.InsertPermission(t.Context(), h.DB.RW(), db.InsertPermissionParams{
		PermissionID: permissionID,
		WorkspaceID:  workspaceID,
		ProjectID:    projectID,
		Name:         "other.permission",
		Slug:         "other.permission",
		Description:  dbtype.NullString{Valid: false},
		CreatedAtM:   time.Now().UnixMilli(),
	}))
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/permissions/%s#read_permission", workspaceID, projectID, permissionID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Permission: permissionID})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, permissionID, res.Body.Data.Id)
}
