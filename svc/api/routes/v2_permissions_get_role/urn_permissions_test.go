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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_get_role"
)

// TestGetRoleAuthorizesCanonicalReadRole guarantees an exact project-scoped
// role grant can read its role.
func TestGetRoleAuthorizesCanonicalReadRole(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspaceID, Name: "canonical-role"})
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/%s#read_role", workspaceID, role.ProjectID, role.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Role: role.ID})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, role.ID, res.Body.Data.Id)
}

// TestGetRoleUsesActualProject guarantees a role grant can read a role in its
// actual project instead of the workspace's default project.
func TestGetRoleUsesActualProject(t *testing.T) {
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
	roleID := uid.New(uid.RolePrefix)
	require.NoError(t, db.Query.InsertRole(t.Context(), h.DB.RW(), db.InsertRoleParams{
		RoleID:      roleID,
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        "other-role",
		Description: sql.NullString{Valid: false},
		CreatedAt:   time.Now().UnixMilli(),
	}))
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/%s#read_role", workspaceID, projectID, roleID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Role: roleID})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Equal(t, roleID, res.Body.Data.Id)
}
