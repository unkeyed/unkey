package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_projects_list_projects"
)

func TestListProjectsSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("empty workspace returns empty list", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Empty(t, res.Body.Data)
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	seeded := map[string]string{}
	for i := 0; i < 3; i++ {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:               uid.New(uid.ProjectPrefix),
			WorkspaceID:      workspace.ID,
			Name:             fmt.Sprintf("Project %d", i),
			Slug:             slug,
			DeleteProtection: i == 0,
		})
		seeded[project.Slug] = project.ID
	}

	t.Run("lists seeded projects with populated fields", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Len(t, res.Body.Data, len(seeded))
		require.False(t, res.Body.Pagination.HasMore)

		protectedCount := 0
		for _, p := range res.Body.Data {
			require.True(t, strings.HasPrefix(p.Id, "proj_"), "id should have proj_ prefix: %s", p.Id)
			require.Equal(t, seeded[p.Slug], p.Id)
			require.NotEmpty(t, p.Name)
			require.Greater(t, p.CreatedAt, int64(0))
			require.Zero(t, p.UpdatedAt, "never-updated project should have zero (omitted) updatedAt")
			if p.DeleteProtection {
				protectedCount++
			}
		}
		require.Equal(t, 1, protectedCount, "exactly one seeded project has deleteProtection=true")
	})

	t.Run("non-existent cursor returns 200 without error", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Cursor: ptr.P("proj_doesnotexist"),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Pagination)
	})
}

func TestListProjectsPagination(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	total := 5
	for i := 0; i < total; i++ {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        fmt.Sprintf("Project %d", i),
			Slug:        slug,
		})
	}

	seen := map[string]struct{}{}
	cursor := (*string)(nil)
	pages := 0
	for {
		req := handler.Request{Limit: ptr.P(2)}
		if cursor != nil {
			req.Cursor = cursor
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.LessOrEqual(t, len(res.Body.Data), 2)

		for _, p := range res.Body.Data {
			_, dup := seen[p.Id]
			require.False(t, dup, "project %s returned on more than one page", p.Id)
			seen[p.Id] = struct{}{}
		}

		pages++
		require.LessOrEqual(t, pages, total+1, "pagination did not terminate")

		if !res.Body.Pagination.HasMore {
			require.Nil(t, res.Body.Pagination.Cursor)
			break
		}
		require.NotNil(t, res.Body.Pagination.Cursor)
		cursor = res.Body.Pagination.Cursor
	}

	require.Len(t, seen, total)
}

func TestListProjectsWorkspaceIsolation(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	mine := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Mine",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
	})

	otherWorkspace := h.CreateWorkspace()
	theirs := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: otherWorkspace.ID,
		Name:        "Theirs",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
	})

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Len(t, res.Body.Data, 1)
	require.Equal(t, mine.ID, res.Body.Data[0].Id)

	t.Run("foreign cursor does not leak across workspaces", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Cursor: ptr.P(theirs.ID),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		for _, p := range res.Body.Data {
			require.NotEqual(t, theirs.ID, p.Id, "foreign project must never be returned")
			require.Equal(t, mine.ID, p.Id, "only own-workspace projects may be returned")
		}
	})

	t.Run("malformed cursor terminates cleanly", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Cursor: ptr.P("not-a-valid-cursor-@@@"),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Pagination)
		for _, p := range res.Body.Data {
			require.Equal(t, mine.ID, p.Id, "only own-workspace projects may be returned")
		}
	})
}

func TestListProjectsSearch(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	// Fresh workspace so search results are not polluted by other tests
	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_project")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	seeded := []struct {
		Name string
		Slug string
	}{
		{"Billing Service", "billing-service"},
		{"Web Frontend", "web-frontend"},
		{"Reports 100%", "reports-full"},
	}
	projectIDs := make(map[string]string)
	for _, p := range seeded {
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        p.Name,
			Slug:        p.Slug,
		})
		projectIDs[p.Name] = project.ID
	}

	list := func(t *testing.T, search string) []string {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Search: ptr.P(search),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		names := make([]string, 0, len(res.Body.Data))
		for _, p := range res.Body.Data {
			names = append(names, p.Name)
		}
		return names
	}

	t.Run("matches name substring", func(t *testing.T) {
		require.Equal(t, []string{"Billing Service"}, list(t, "billing"))
	})

	t.Run("matches slug substring", func(t *testing.T) {
		require.Equal(t, []string{"Web Frontend"}, list(t, "web-frontend"))
	})

	t.Run("matches project id", func(t *testing.T) {
		require.Equal(t, []string{"Reports 100%"}, list(t, projectIDs["Reports 100%"]))
	})

	t.Run("is case insensitive", func(t *testing.T) {
		require.Equal(t, []string{"Billing Service"}, list(t, "BILLING"))
	})

	t.Run("wildcards match literally", func(t *testing.T) {
		// Unescaped, "%" would match every project and "s_rvice" would match "Service"
		require.Equal(t, []string{"Reports 100%"}, list(t, "100%"))
		require.Empty(t, list(t, "s_rvice"))
	})

	t.Run("no matches returns empty list", func(t *testing.T) {
		require.Empty(t, list(t, "does-not-exist"))
	})

	t.Run("whitespace-only search returns all", func(t *testing.T) {
		require.ElementsMatch(t,
			[]string{"Billing Service", "Web Frontend", "Reports 100%"},
			list(t, "   "),
		)
	})
}
