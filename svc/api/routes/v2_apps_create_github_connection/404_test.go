package handler_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_create_github_connection"
)

func TestCreateGithubConnectionNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "app.*.connect_github")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	t.Run("unknown app returns 404", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: uid.New(uid.ProjectPrefix),
			App:     uid.New(uid.AppPrefix),
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	// Security: a caller must not be able to mint an install state for an app in
	// another workspace (which would let them bind an installation there).
	t.Run("app in another workspace returns 404", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		otherProjectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		otherProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "Theirs",
			Slug:        otherProjectSlug,
		})
		otherAppSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		otherApp := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   otherWorkspace.ID,
			ProjectID:     otherProject.ID,
			Name:          "Theirs",
			Slug:          otherAppSlug,
			DefaultBranch: "main",
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: otherProject.ID,
			App:     otherApp.ID,
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace app, received: %s", res.RawBody)
		require.NotContains(t, res.RawBody, otherApp.ID, "response must not leak the other workspace's app ID")
	})

	t.Run("app id with mismatched project returns 404", func(t *testing.T) {
		projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		project := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: workspace.ID,
			Name:        "Mine",
			Slug:        projectSlug,
		})
		appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		app := h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   workspace.ID,
			ProjectID:     project.ID,
			Name:          "Mine",
			Slug:          appSlug,
			DefaultBranch: "main",
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
			Project: uid.New(uid.ProjectPrefix),
			App:     app.ID,
		})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 when app does not belong to project, received: %s", res.RawBody)
	})
}

// The existence oracle must stay closed: an unauthorized same-workspace key
// querying a real app and a nonexistent app gets byte-identical error bodies
// (modulo the request id), so it cannot enumerate the workspace's apps.
func TestCreateGithubConnectionExistenceNotLeaked(t *testing.T) {
	h := testutil.NewHarness(t)

	route := newRoute(h)
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})
	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	missingID := uid.New(uid.AppPrefix)

	// Same-workspace key without the connect_github action.
	rootKey := h.CreateRootKey(workspace.ID, "api.*.read_api")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	realRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.ID, App: app.ID})
	missingRes := testutil.CallRoute[handler.Request, openapi.NotFoundErrorResponse](h, route, headers, handler.Request{Project: project.ID, App: missingID})

	require.Equal(t, http.StatusNotFound, realRes.Status, "real app should look not-found, got: %s", realRes.RawBody)
	require.Equal(t, http.StatusNotFound, missingRes.Status, "missing app should be not-found, got: %s", missingRes.RawBody)
	require.NotContains(t, realRes.RawBody, app.ID, "response must not leak the app ID")

	require.Equal(t, missingRes.Body.Error.Detail, realRes.Body.Error.Detail, "error detail must be identical for real and missing apps")
	require.Equal(t, missingRes.Body.Error.Type, realRes.Body.Error.Type, "error type must be identical for real and missing apps")
	require.Equal(t, missingRes.Body.Error.Status, realRes.Body.Error.Status, "error status must be identical for real and missing apps")
}
