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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_list_apps"
)

func TestListAppsSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.read_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})

	t.Run("project with no apps returns empty list", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug})
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
		app := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("App %d", i),
			Slug:          slug,
			DefaultBranch: "main",
		})
		seeded[app.ID] = slug
	}

	t.Run("lists seeded apps by project id", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.ID})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, len(seeded))
		for _, a := range res.Body.Data {
			_, ok := seeded[a.Id]
			require.True(t, ok, "unexpected app %s in response", a.Id)
		}
	})

	t.Run("lists seeded apps with populated fields", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Len(t, res.Body.Data, len(seeded))
		require.False(t, res.Body.Pagination.HasMore)

		for _, a := range res.Body.Data {
			slug, ok := seeded[a.Id]
			require.True(t, ok, "unexpected app %s in response", a.Id)
			require.True(t, strings.HasPrefix(a.Id, "app_"), "id should have app_ prefix: %s", a.Id)
			require.Equal(t, slug, a.Slug)
			require.Equal(t, project.ID, a.ProjectId)
			require.Equal(t, "main", a.DefaultBranch)
			require.NotEmpty(t, a.Name)
			require.Empty(t, a.CurrentDeploymentId, "freshly seeded app has no active deployment")
			require.False(t, a.IsRolledBack)
			require.False(t, a.DeleteProtection)
			require.Greater(t, a.CreatedAt, int64(0))
			require.Zero(t, a.UpdatedAt, "never-updated app should have zero (omitted) updatedAt")
		}
	})

	t.Run("apps from other projects are not listed", func(t *testing.T) {
		otherSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		otherProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Billing Service",
			Slug:        otherSlug,
		})
		strayApp := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     otherProject.ID,
			Name:          "Stray",
			Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
			DefaultBranch: "main",
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, len(seeded))
		for _, a := range res.Body.Data {
			require.NotEqual(t, strayApp.ID, a.Id, "app from another project must not be listed")
		}
	})

	t.Run("non-existent cursor returns 200 without error", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.Slug,
			Cursor:  ptr.P("app_doesnotexist"),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Pagination)
	})

	t.Run("cursor borrowed from another workspace stays project-scoped", func(t *testing.T) {
		// A caller-supplied cursor that is a real app ID from another workspace
		// must not widen results beyond the authorized project. The query is
		// scoped by project_id, so the foreign app can never appear and only this
		// project's own apps (with id >= cursor) are returned.
		foreignWorkspace := h.CreateWorkspace()
		foreignSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		foreignProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: foreignWorkspace.ID,
			Name:        "Foreign",
			Slug:        foreignSlug,
		})
		foreignApp := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   foreignWorkspace.ID,
			ProjectID:     foreignProject.ID,
			Name:          "Foreign App",
			Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
			DefaultBranch: "main",
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.Slug,
			Cursor:  ptr.P(foreignApp.ID),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		for _, a := range res.Body.Data {
			require.NotEqual(t, foreignApp.ID, a.Id, "foreign app must never be returned")
			require.Equal(t, project.ID, a.ProjectId, "only the authorized project's apps may be returned")
		}
		require.NotContains(t, res.RawBody, foreignProject.ID, "response must not leak the foreign project ID")
	})
}

func TestListAppsPagination(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.read_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})

	total := 5
	for i := 0; i < total; i++ {
		h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          fmt.Sprintf("App %d", i),
			Slug:          strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-")),
			DefaultBranch: "main",
		})
	}

	seen := map[string]struct{}{}
	cursor := (*string)(nil)
	pages := 0
	for {
		req := handler.Request{Project: project.Slug, Limit: ptr.P(2)}
		if cursor != nil {
			req.Cursor = cursor
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.LessOrEqual(t, len(res.Body.Data), 2)

		for _, a := range res.Body.Data {
			_, dup := seen[a.Id]
			require.False(t, dup, "app %s returned on more than one page", a.Id)
			seen[a.Id] = struct{}{}
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
