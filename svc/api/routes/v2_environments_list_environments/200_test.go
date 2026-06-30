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
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_environment")
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

			// Settings are returned inline, same as getEnvironment.
			// CreateEnvironment seeds default runtime and build settings.
			require.NotNil(t, e.CpuMillicores)
			require.Equal(t, 100, *e.CpuMillicores)
			require.NotNil(t, e.MemoryMib)
			require.Equal(t, 128, *e.MemoryMib)
			require.NotNil(t, e.RootDirectory)
			require.Equal(t, ".", *e.RootDirectory)
			require.NotNil(t, e.AutoDeploy)
			require.True(t, *e.AutoDeploy)
			require.Nil(t, e.Regions, "no regional settings seeded")
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
}
