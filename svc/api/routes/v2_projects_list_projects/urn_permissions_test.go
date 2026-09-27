package handler_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_list_projects"
)

// TestListProjectsRefillsAuthorizedPages guarantees specific-project permissions
// produce full pages without exposing denied IDs. For example, permission for
// the fourth and seventh projects returns those two across one-item pages,
// while a search matching only the first project returns an empty page.
func TestListProjectsRefillsAuthorizedPages(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)
	workspace := h.CreateWorkspace()
	ids := make([]string, 8)
	for i := range ids {
		ids[i] = h.CreateProject(seed.CreateProjectRequest{
			ID: strings.ToLower(uid.New(uid.ProjectPrefix)), WorkspaceID: workspace.ID,
			Name: "Visible project", Slug: uid.New("project"),
		}).ID
	}
	slices.Sort(ids)
	key := h.CreateRootKey(workspace.ID,
		fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(ids[3])),
		fmt.Sprintf("%s#read", urn.New().Workspace(workspace.ID).Project(ids[6])),
	)
	headers := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + key}}
	first := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: ptr.P(1)})
	require.Equal(t, http.StatusOK, first.Status, "%s", first.RawBody)
	require.Len(t, first.Body.Data, 1)
	require.Equal(t, ids[3], first.Body.Data[0].Id)
	require.True(t, first.Body.Pagination.HasMore)
	require.Equal(t, ptr.P(ids[6]), first.Body.Pagination.Cursor)
	last := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Limit: ptr.P(1), Cursor: first.Body.Pagination.Cursor})
	require.Equal(t, http.StatusOK, last.Status, "%s", last.RawBody)
	require.Len(t, last.Body.Data, 1)
	require.Equal(t, ids[6], last.Body.Data[0].Id)
	require.False(t, last.Body.Pagination.HasMore)
	require.Nil(t, last.Body.Pagination.Cursor)
	for _, i := range []int{0, 1, 2, 4, 5, 7} {
		require.NotContains(t, first.RawBody, ids[i])
		require.NotContains(t, last.RawBody, ids[i])
	}
	empty := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Search: ptr.P(ids[0])})
	require.Equal(t, http.StatusOK, empty.Status, "%s", empty.RawBody)
	require.Empty(t, empty.Body.Data)
	require.False(t, empty.Body.Pagination.HasMore)
	require.Nil(t, empty.Body.Pagination.Cursor)
}

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

// TestListProjectsFiltersInsufficientPermissions guarantees a wrong action or
// workspace returns an empty page. For example, projects/*#write cannot reveal
// a project's ID or produce a pagination cursor.
func TestListProjectsFiltersInsufficientPermissions(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.CreateWorkspace()
	h.CreateProject(seed.CreateProjectRequest{
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
			require.Equal(t, http.StatusOK, res.Status, "received: %s", res.RawBody)
			require.Empty(t, res.Body.Data)
			require.Nil(t, res.Body.Pagination.Cursor)
			require.False(t, res.Body.Pagination.HasMore)
		})
	}
}
