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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_permissions_list_roles"
)

// TestListRolesAuthorizesCanonicalReadRole guarantees a wildcard project-scoped
// role grant can list roles in its project.
func TestListRolesAuthorizesCanonicalReadRole(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	role := h.CreateRole(seed.CreateRoleRequest{WorkspaceID: workspaceID, Name: "canonical-role"})
	otherProjectID := uid.New(uid.ProjectPrefix)
	h.CreateProject(seed.CreateProjectRequest{
		ID:          otherProjectID,
		WorkspaceID: workspaceID,
		Name:        "Other",
		Slug:        uid.New("other"),
	})
	require.NoError(t, db.Query.InsertRole(t.Context(), h.DB.RW(), db.InsertRoleParams{
		RoleID:      uid.New(uid.RolePrefix),
		WorkspaceID: workspaceID,
		ProjectID:   otherProjectID,
		Name:        "other-role",
		Description: sql.NullString{Valid: false},
		CreatedAt:   time.Now().UnixMilli(),
	}))
	rootKey := h.CreateRootKey(
		workspaceID,
		fmt.Sprintf("unkey:v1:%s:projects/%s/rbac/roles/*#read", workspaceID, role.ProjectID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, role.ID, res.Body.Data[0].Id)
}

// TestListRolesAuthorizesCanonicalReadBeforeDefaultProjectCreation guarantees
// that a project-wildcard grant can authorize a missing default project before
// the route creates it.
func TestListRolesAuthorizesCanonicalReadBeforeDefaultProjectCreation(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(
		workspace.ID,
		fmt.Sprintf("unkey:v1:%s:projects/*/rbac/roles/*#read", workspace.ID),
	)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	require.Empty(t, res.Body.Data)
}
