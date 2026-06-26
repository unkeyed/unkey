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

func TestGetAppSuccessfully(t *testing.T) {
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

	appSlug := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
	app := h.CreateApp(seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspace.ID,
		ProjectID:     project.ID,
		Name:          "Payments API",
		Slug:          appSlug,
		DefaultBranch: "main",
	})

	for _, tc := range []struct {
		name    string
		project string
		app     string
	}{
		{name: "by ids", project: project.ID, app: app.ID},
		{name: "by slugs", project: projectSlug, app: appSlug},
		{name: "project id and app slug", project: project.ID, app: appSlug},
		{name: "project slug and app id", project: projectSlug, app: app.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project: tc.project,
				App:     tc.app,
			})
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, app.ID, res.Body.Data.Id)
			require.True(t, strings.HasPrefix(res.Body.Data.Id, "app_"), "id should have app_ prefix: %s", res.Body.Data.Id)
			require.Equal(t, "Payments API", res.Body.Data.Name)
			require.Equal(t, appSlug, res.Body.Data.Slug)
			require.Equal(t, project.ID, res.Body.Data.ProjectId)
			require.Equal(t, "main", res.Body.Data.DefaultBranch)
			require.Empty(t, res.Body.Data.CurrentDeploymentId, "freshly seeded app has no active deployment")
			require.False(t, res.Body.Data.IsRolledBack)
			require.False(t, res.Body.Data.DeleteProtection)
			require.Greater(t, res.Body.Data.CreatedAt, int64(0))
			require.Zero(t, res.Body.Data.UpdatedAt, "never-updated app should have zero (omitted) updatedAt")
		})
	}
}
