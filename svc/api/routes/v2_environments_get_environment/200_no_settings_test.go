package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_get_environment"
)

func TestGetEnvironmentWithoutSettings(t *testing.T) {
	ctx := context.Background()
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB}
	h.Register(route)

	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.read_environment")
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

	// CreateEnvironment seeds default settings; drop them to simulate an
	// environment that has not been deployed yet.
	require.NoError(t, db.Query.DeleteAppRuntimeSettingsByEnvironmentId(ctx, h.DB.RW(), environment.ID))
	require.NoError(t, db.Query.DeleteAppBuildSettingsByEnvironmentId(ctx, h.DB.RW(), environment.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
		Project:     project.ID,
		App:         app.ID,
		Environment: environment.ID,
	})
	require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, environment.ID, res.Body.Data.Id)

	// Base fields stay populated.
	require.Equal(t, "production", res.Body.Data.Slug)
	require.Greater(t, res.Body.Data.CreatedAt, int64(0))

	// Runtime settings omitted.
	require.Nil(t, res.Body.Data.Port)
	require.Nil(t, res.Body.Data.CpuMillicores)
	require.Nil(t, res.Body.Data.MemoryMib)
	require.Nil(t, res.Body.Data.StorageMib)
	require.Nil(t, res.Body.Data.Command)
	require.Nil(t, res.Body.Data.Healthcheck)
	require.Nil(t, res.Body.Data.ShutdownSignal)
	require.Nil(t, res.Body.Data.UpstreamProtocol)
	require.Nil(t, res.Body.Data.OpenapiSpecPath)

	// Build settings omitted.
	require.Nil(t, res.Body.Data.Dockerfile)
	require.Nil(t, res.Body.Data.RootDirectory)
	require.Nil(t, res.Body.Data.WatchPaths)
	require.Nil(t, res.Body.Data.AutoDeploy)

	// No regional settings seeded.
	require.Nil(t, res.Body.Data.Regions)
}
