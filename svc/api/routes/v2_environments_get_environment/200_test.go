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
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_get_environment"
)

func TestGetEnvironmentSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "project.*.read_environment")
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

	environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspace.ID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "production",
		Description: "Production environment",
	})

	for _, tc := range []struct {
		name        string
		project     string
		app         string
		environment string
	}{
		{name: "by ids", project: project.ID, app: app.ID, environment: environment.ID},
		{name: "by slugs", project: projectSlug, app: appSlug, environment: "production"},
		{name: "mixed ids and slugs", project: project.ID, app: appSlug, environment: environment.ID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project:     tc.project,
				App:         tc.app,
				Environment: tc.environment,
			})
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.Equal(t, environment.ID, res.Body.Data.Id)
			require.True(t, strings.HasPrefix(res.Body.Data.Id, "env_"), "id should have env_ prefix: %s", res.Body.Data.Id)
			require.Equal(t, project.ID, res.Body.Data.ProjectId)
			require.Equal(t, app.ID, res.Body.Data.AppId)
			require.Equal(t, "production", res.Body.Data.Slug)
			require.Equal(t, "Production environment", res.Body.Data.Description)
			require.False(t, res.Body.Data.DeleteProtection)
			require.Greater(t, res.Body.Data.CreatedAt, int64(0))
			require.Zero(t, res.Body.Data.UpdatedAt, "never-updated environment should have zero (omitted) updatedAt")
		})
	}
}
