package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apis_create_api"
)

// TestCreateApiAuthorizesURNKeyspaceWrite guarantees that API creation
// accepts a keyspace write permission for the workspace's default project.
func TestCreateApiAuthorizesURNKeyspaceWrite(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	projectID, err := projects.EnsureDefaultProject(t.Context(), h.DB.RW(), workspaceID)
	require.NoError(t, err)
	rootKey := h.CreateRootKey(
		workspaceID,
		rbac.U(urn.New().Workspace(workspaceID).Project(projectID).Keyspace("*"), permissions.Write).Value,
	)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Name: "URN-api"})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	api, err := db.Query.FindApiByID(t.Context(), h.DB.RO(), res.Body.Data.ApiId)
	require.NoError(t, err)
	require.Equal(t, projectID, api.ProjectID)
}

// TestCreateApiAuthorizesProjectWildcardBeforeDefaultProjectCreation guarantees
// that authorization happens before a missing default project is created.
func TestCreateApiAuthorizesProjectWildcardBeforeDefaultProjectCreation(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(
		workspace.ID,
		rbac.U(urn.New().Workspace(workspace.ID).Project("*").Keyspace("*"), permissions.Write).Value,
	)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Name: "project-wildcard-api"})

	require.Equal(t, http.StatusOK, res.Status, res.RawBody)
	projectID, found, err := projects.FindDefaultProject(t.Context(), h.DB.RO(), workspace.ID)
	require.NoError(t, err)
	require.True(t, found)
	api, err := db.Query.FindApiByID(t.Context(), h.DB.RO(), res.Body.Data.ApiId)
	require.NoError(t, err)
	require.Equal(t, projectID, api.ProjectID)
}

// TestCreateApiURNPermissionBoundaries guarantees that URN permissions
// preserve workspace, project, and action boundaries while supporting broader
// catalog wildcards.
func TestCreateApiURNPermissionBoundaries(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspaceID := h.Resources().UserWorkspace.ID
	projectID, err := projects.EnsureDefaultProject(t.Context(), h.DB.RW(), workspaceID)
	require.NoError(t, err)
	otherWorkspace := h.CreateWorkspace()
	otherProjectID := uid.New(uid.ProjectPrefix)
	project := urn.New().Workspace(workspaceID).Project(projectID)

	tests := []struct {
		name       string
		permission string
		status     int
	}{
		{
			name:       "project and keyspace wildcards",
			permission: rbac.U(urn.New().Workspace(workspaceID).Project("*").Keyspace("*"), permissions.Write).Value,
			status:     http.StatusOK,
		},
		{
			name: "project subtree wildcard",
			permission: rbac.U(urn.V1{
				WorkspaceID: project.WorkspaceID,
				Resource:    "projects/" + project.ProjectID + "/**",
			}, permissions.Write).Value,
			status: http.StatusOK,
		},
		{
			name: "workspace admin wildcard",
			permission: rbac.U(urn.V1{
				WorkspaceID: workspaceID,
				Resource:    "**",
			}, urn.PermissionAll).Value,
			status: http.StatusOK,
		},
		{
			name:       "wrong project",
			permission: rbac.U(urn.New().Workspace(workspaceID).Project(otherProjectID).Keyspace("*"), permissions.Write).Value,
			status:     http.StatusForbidden,
		},
		{
			name:       "wrong workspace",
			permission: rbac.U(urn.New().Workspace(otherWorkspace.ID).Project(projectID).Keyspace("*"), permissions.Write).Value,
			status:     http.StatusForbidden,
		},
		{
			name:       "wrong action",
			permission: rbac.U(urn.New().Workspace(workspaceID).Project(projectID).Keyspace("*"), permissions.Read).Value,
			status:     http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspaceID, test.permission)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}, handler.Request{Name: "permission-boundary-api"})

			require.Equal(t, test.status, res.Status, res.RawBody)
		})
	}
}

// TestCreateApiAuthorizationFailureDoesNotCreateDefaultProject guarantees that
// resolving authorization scope cannot mutate project state for a denied call.
func TestCreateApiAuthorizationFailureDoesNotCreateDefaultProject(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(
		workspace.ID,
		rbac.U(urn.New().Workspace(workspace.ID).Project("*").Keyspace("*"), permissions.Read).Value,
	)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}, handler.Request{Name: "denied-api"})

	require.Equal(t, http.StatusForbidden, res.Status, res.RawBody)
	_, found, err := projects.FindDefaultProject(t.Context(), h.DB.RO(), workspace.ID)
	require.NoError(t, err)
	require.False(t, found)
}
