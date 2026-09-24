package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_list_projects"
)

// TestListProjectsAuthorizesCollectionURN guarantees a collection URN can list
// projects only from its authorized workspace without a legacy tuple permission.
func TestListProjectsAuthorizesCollectionURN(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Mine",
		Slug:        uid.New("mine"),
	})
	otherWorkspace := h.CreateWorkspace()
	h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: otherWorkspace.ID,
		Name:        "Theirs",
		Slug:        uid.New("theirs"),
	})

	permission := fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project("*"))
	rootKey := h.CreateRootKey(workspace.ID, permission)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, project.ID, res.Body.Data[0].Id)
}

// TestListProjectsAuthorizesCollectionURNForEmptyList guarantees authorization
// does not depend on finding a project to check.
func TestListProjectsAuthorizesCollectionURNForEmptyList(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	permission := fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project("*"))
	rootKey := h.CreateRootKey(workspace.ID, permission)
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "received: %s", res.RawBody)
	require.Empty(t, res.Body.Data)
}

// TestListProjectsRejectsInsufficientURNs guarantees a workspace list cannot be
// authorized by a concrete project, another action, or another workspace.
func TestListProjectsRejectsInsufficientURNs(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Project",
		Slug:        uid.New("project"),
	})
	otherWorkspace := h.CreateWorkspace()

	testCases := []struct {
		name       string
		permission string
	}{
		{
			name:       "concrete project",
			permission: fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(project.ID)),
		},
		{
			name:       "wrong action",
			permission: fmt.Sprintf("%s#write", urn.New().Workspace(workspace.ID).Project("*")),
		},
		{
			name:       "wrong workspace",
			permission: fmt.Sprintf("%s#read", urn.New().Workspace(otherWorkspace.ID).Project("*")),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rootKey := h.CreateRootKey(workspace.ID, testCase.permission)
			headers := http.Header{
				"Content-Type":  {"application/json"},
				"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
			}

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
			require.Equal(t, http.StatusForbidden, res.Status, "received: %s", res.RawBody)
		})
	}
}
