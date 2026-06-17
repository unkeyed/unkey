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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_apps_get_app"
)

func TestGetAppNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_app")
	headers := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
	}

	projectSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        projectSlug,
	})

	t.Run("unknown app slug returns 404", func(t *testing.T) {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ProjectSlug: projectSlug, Slug: slug})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("unknown project slug returns 404", func(t *testing.T) {
		slug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ProjectSlug: slug, Slug: "app-slug"})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

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
		h.CreateApp(seed.CreateAppRequest{
			ID:            uid.New(uid.AppPrefix),
			WorkspaceID:   otherWorkspace.ID,
			ProjectID:     otherProject.ID,
			Name:          "Theirs",
			Slug:          otherAppSlug,
			DefaultBranch: "main",
		})

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{ProjectSlug: otherProjectSlug, Slug: otherAppSlug})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace app, received: %s", res.RawBody)
	})
}
