package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_get_environment"
)

func TestGetEnvironment(t *testing.T) {
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

	t.Run("with default settings", func(t *testing.T) {
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

				require.NotNil(t, res.Body.Data.Port)
				require.Equal(t, 8080, *res.Body.Data.Port)
				require.NotNil(t, res.Body.Data.CpuMillicores)
				require.Equal(t, 100, *res.Body.Data.CpuMillicores)
				require.NotNil(t, res.Body.Data.MemoryMib)
				require.Equal(t, 128, *res.Body.Data.MemoryMib)
				require.NotNil(t, res.Body.Data.StorageMib)
				require.Equal(t, 0, *res.Body.Data.StorageMib)
				require.NotNil(t, res.Body.Data.ShutdownSignal)
				require.EqualValues(t, "SIGTERM", *res.Body.Data.ShutdownSignal)
				require.NotNil(t, res.Body.Data.UpstreamProtocol)
				require.EqualValues(t, "http1", *res.Body.Data.UpstreamProtocol)
				require.Nil(t, res.Body.Data.Healthcheck, "no healthcheck configured")
				require.Nil(t, res.Body.Data.OpenapiSpecPath, "no openapi spec path configured")

				require.NotNil(t, res.Body.Data.Dockerfile)
				require.Equal(t, "Dockerfile", *res.Body.Data.Dockerfile)
				require.NotNil(t, res.Body.Data.RootDirectory)
				require.Equal(t, ".", *res.Body.Data.RootDirectory)
				require.NotNil(t, res.Body.Data.AutoDeploy)
				require.True(t, *res.Body.Data.AutoDeploy)
				require.Nil(t, res.Body.Data.BuildCommand, "no build command configured")

				require.Nil(t, res.Body.Data.Regions)
			})
		}
	})

	t.Run("without settings", func(t *testing.T) {
		environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:          uid.New(uid.EnvironmentPrefix),
			WorkspaceID: workspace.ID,
			ProjectID:   project.ID,
			AppID:       app.ID,
			Slug:        "no-settings",
			Description: "Not yet deployed",
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

		require.Equal(t, "no-settings", res.Body.Data.Slug)
		require.Greater(t, res.Body.Data.CreatedAt, int64(0))

		require.Nil(t, res.Body.Data.Port)
		require.Nil(t, res.Body.Data.CpuMillicores)
		require.Nil(t, res.Body.Data.MemoryMib)
		require.Nil(t, res.Body.Data.StorageMib)
		require.Nil(t, res.Body.Data.Command)
		require.Nil(t, res.Body.Data.Healthcheck)
		require.Nil(t, res.Body.Data.ShutdownSignal)
		require.Nil(t, res.Body.Data.UpstreamProtocol)
		require.Nil(t, res.Body.Data.OpenapiSpecPath)

		require.Nil(t, res.Body.Data.Dockerfile)
		require.Nil(t, res.Body.Data.RootDirectory)
		require.Nil(t, res.Body.Data.BuildCommand)
		require.Nil(t, res.Body.Data.WatchPaths)
		require.Nil(t, res.Body.Data.AutoDeploy)

		require.Nil(t, res.Body.Data.Regions)
	})

	t.Run("regions", func(t *testing.T) {
		platform := strings.ToLower(strings.ReplaceAll(uid.New("test"), "_", "-"))
		require.NoError(t, db.Query.UpsertRegion(ctx, h.DB.RW(), db.UpsertRegionParams{
			ID:       uid.New(uid.RegionPrefix),
			Name:     "us-east-1",
			Platform: platform,
		}))
		region, err := db.Query.FindRegionByPlatformAndName(ctx, h.DB.RO(), db.FindRegionByPlatformAndNameParams{
			Name:     "us-east-1",
			Platform: platform,
		})
		require.NoError(t, err)
		regionID := region.ID

		now := time.Now().UnixMilli()

		t.Run("without autoscaling policy uses replica count as min and max", func(t *testing.T) {
			environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
				ID:          uid.New(uid.EnvironmentPrefix),
				WorkspaceID: workspace.ID,
				ProjectID:   project.ID,
				AppID:       app.ID,
				Slug:        "regions-static",
				Description: "Static replicas",
			})

			require.NoError(t, db.Query.UpsertAppRegionalSettings(ctx, h.DB.RW(), db.UpsertAppRegionalSettingsParams{
				WorkspaceID:   workspace.ID,
				AppID:         app.ID,
				EnvironmentID: environment.ID,
				RegionID:      regionID,
				Replicas:      3,
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: false},
			}))

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project:     project.ID,
				App:         app.ID,
				Environment: environment.ID,
			})
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.NotNil(t, res.Body.Data.Regions)
			regions := *res.Body.Data.Regions
			require.Len(t, regions, 1)
			require.Equal(t, "us-east-1", regions[0].Name)
			require.Equal(t, 3, regions[0].Replicas.Min)
			require.Equal(t, 3, regions[0].Replicas.Max)
		})

		t.Run("with autoscaling policy uses policy bounds", func(t *testing.T) {
			environment := h.CreateEnvironment(seed.CreateEnvironmentRequest{
				ID:          uid.New(uid.EnvironmentPrefix),
				WorkspaceID: workspace.ID,
				ProjectID:   project.ID,
				AppID:       app.ID,
				Slug:        "regions-autoscaling",
				Description: "Autoscaling replicas",
			})

			require.NoError(t, db.Query.UpsertAppRegionalSettings(ctx, h.DB.RW(), db.UpsertAppRegionalSettingsParams{
				WorkspaceID:   workspace.ID,
				AppID:         app.ID,
				EnvironmentID: environment.ID,
				RegionID:      regionID,
				Replicas:      2,
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: false},
			}))

			// No typed query for autoscaling policies on this branch; insert raw and
			// point the regional setting at it.
			policyID := uid.New("hap")
			_, err := h.DB.RW().ExecContext(ctx,
				"INSERT INTO horizontal_autoscaling_policies (id, workspace_id, replicas_min, replicas_max, cpu_threshold, memory_threshold, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
				policyID, workspace.ID, 1, 5, 80, 80, now)
			require.NoError(t, err)
			_, err = h.DB.RW().ExecContext(ctx,
				"UPDATE app_regional_settings SET horizontal_autoscaling_policy_id = ? WHERE app_id = ? AND environment_id = ? AND region_id = ?",
				policyID, app.ID, environment.ID, regionID)
			require.NoError(t, err)

			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, handler.Request{
				Project:     project.ID,
				App:         app.ID,
				Environment: environment.ID,
			})
			require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
			require.NotNil(t, res.Body.Data.Regions)
			regions := *res.Body.Data.Regions
			require.Len(t, regions, 1)
			require.Equal(t, "us-east-1", regions[0].Name)
			require.Equal(t, 1, regions[0].Replicas.Min)
			require.Equal(t, 5, regions[0].Replicas.Max)
		})
	})
}
