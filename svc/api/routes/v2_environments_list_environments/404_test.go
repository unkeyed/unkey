package handler_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_list_environments"
)

func TestListEnvironmentsNotFound(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace

	project := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspace.ID,
		Name:        "Payments Service",
		Slug:        slug(t),
	})
	app := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		Name:        "Payments API",
		Slug:        slug(t),
	})

	t.Run("unknown app slug returns 404", func(t *testing.T) {
		missingApp := slug(t)
		rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/*#read", workspace.ID, project.ID, missingApp))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: project.Slug, App: missingApp})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("unknown project slug returns 404", func(t *testing.T) {
		missingProject := slug(t)
		rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/*#read", workspace.ID, missingProject, app.ID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: missingProject, App: app.Slug})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404, received: %s", res.RawBody)
	})

	t.Run("app in another workspace returns 404", func(t *testing.T) {
		otherWorkspace := h.CreateWorkspace()
		otherProject := h.CreateProject(seed.CreateProjectRequest{
			ID:          uid.New(uid.ProjectPrefix),
			WorkspaceID: otherWorkspace.ID,
			Name:        "Theirs",
			Slug:        slug(t),
		})
		otherApp := h.CreateApp(seed.CreateAppRequest{
			ID:          uid.New(uid.AppPrefix),
			WorkspaceID: otherWorkspace.ID,
			ProjectID:   otherProject.ID,
			Name:        "Theirs",
			Slug:        slug(t),
		})
		h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: otherWorkspace.ID,
			ProjectID:   otherProject.ID,
			AppID:       otherApp.ID,
			Slug:        slug(t),
		})

		rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:projects/%s/apps/%s/environments/*#read", workspace.ID, otherProject.ID, otherApp.ID))
		headers := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {fmt.Sprintf("Bearer %s", rootKey)},
		}

		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{Project: otherProject.Slug, App: otherApp.Slug})
		require.Equal(t, http.StatusNotFound, res.Status, "expected 404 for cross-workspace app, received: %s", res.RawBody)
	})
}
