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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environments"
)

func slug(t *testing.T) string {
	t.Helper()
	return strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
}

func TestListEnvironmentsSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_environment")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        slug(t),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          slug(t),
		DefaultBranch: "main",
	})

	t.Run("app with no environments returns empty list", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug, App: app.Slug})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body)
		require.Empty(t, res.Body.Data)
		require.NotNil(t, res.Body.Pagination)
		require.False(t, res.Body.Pagination.HasMore)
		require.Nil(t, res.Body.Pagination.Cursor)
	})

	seeded := map[string]string{}
	for i := 0; i < 3; i++ {
		envSlug := slug(t)
		env := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        envSlug,
			Description: fmt.Sprintf("Environment %d", i),
		})
		seeded[env.ID] = envSlug
	}

	t.Run("lists seeded environments by ids", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.ID, App: app.ID})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, len(seeded))
		for _, e := range res.Body.Data {
			_, ok := seeded[e.Id]
			require.True(t, ok, "unexpected environment %s in response", e.Id)
		}
	})

	t.Run("lists seeded environments with populated fields", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug, App: app.Slug})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
		require.Len(t, res.Body.Data, len(seeded))
		require.False(t, res.Body.Pagination.HasMore)

		for _, e := range res.Body.Data {
			envSlug, ok := seeded[e.Id]
			require.True(t, ok, "unexpected environment %s in response", e.Id)
			require.True(t, strings.HasPrefix(e.Id, "env_"), "id should have env_ prefix: %s", e.Id)
			require.Equal(t, envSlug, e.Slug)
			require.Equal(t, project.ID, e.ProjectId)
			require.Equal(t, app.ID, e.AppId)
			require.NotEmpty(t, e.Description)
			require.False(t, e.DeleteProtection)
			require.Greater(t, e.CreatedAt, int64(0))
		}
	})

	t.Run("environments from another app are not listed", func(t *testing.T) {
		otherApp := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          "Billing API",
			Slug:          slug(t),
			DefaultBranch: "main",
		})
		strayEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       otherApp.ID,
			Slug:        slug(t),
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug, App: app.Slug})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.Len(t, res.Body.Data, len(seeded))
		for _, e := range res.Body.Data {
			require.NotEqual(t, strayEnv.ID, e.Id, "environment from another app must not be listed")
		}
	})

	t.Run("non-existent cursor returns 200 without error", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.Slug,
			App:     app.Slug,
			Cursor:  ptr.P("env_doesnotexist"),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotNil(t, res.Body.Pagination)
	})

	t.Run("cursor borrowed from another app stays app-scoped", func(t *testing.T) {
		// A caller-supplied cursor that is a real environment ID from another app
		// must not widen results beyond the authorized app. The query is scoped by
		// app_id, so the foreign environment can never appear and only this app's
		// own environments (with id >= cursor) are returned.
		foreignApp := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          "Foreign App",
			Slug:          slug(t),
			DefaultBranch: "main",
		})
		foreignEnv := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       foreignApp.ID,
			Slug:        slug(t),
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: project.Slug,
			App:     app.Slug,
			Cursor:  ptr.P(foreignEnv.ID),
		})
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		for _, e := range res.Body.Data {
			require.NotEqual(t, foreignEnv.ID, e.Id, "foreign environment must never be returned")
			require.Equal(t, app.ID, e.AppId, "only the authorized app's environments may be returned")
		}
	})
}

func TestListEnvironmentsPagination(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_environment")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        slug(t),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          slug(t),
		DefaultBranch: "main",
	})

	total := 5
	for i := 0; i < total; i++ {
		h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        slug(t),
		})
	}

	seen := map[string]struct{}{}
	cursor := (*string)(nil)
	pages := 0
	for {
		req := handler.Request{Project: project.Slug, App: app.Slug, Limit: ptr.P(2)}
		if cursor != nil {
			req.Cursor = cursor
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.LessOrEqual(t, len(res.Body.Data), 2)

		for _, e := range res.Body.Data {
			_, dup := seen[e.Id]
			require.False(t, dup, "environment %s returned on more than one page", e.Id)
			seen[e.Id] = struct{}{}
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
