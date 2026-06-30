package handler_test

import (
	"context"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_update_settings"
)

func TestUpdateSettingsSuccessfully(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, QuotaCache: h.Caches.WorkspaceQuota}
	h.Register(route)

	ctx := context.Background()
	workspace := h.Resources().UserWorkspace
	rootKey := h.CreateRootKey(workspace.ID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	// Seed regions used by the region reconciliation subtests.
	seedRegions(t, h, "us-east-1", "us-west-2")

	call := func(t *testing.T, req handler.Request) {
		t.Helper()
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers, req)
		require.Equal(t, 200, res.Status, "expected 200, received: %s", res.RawBody)
		require.NotEmpty(t, res.Body.Meta.RequestId)
	}

	t.Run("build settings", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project:       env.projectID,
			App:           env.appID,
			Environment:   env.environmentID,
			Dockerfile:    nullable.NewNullableWithValue("Dockerfile.prod"),
			RootDirectory: ptr("./app"),
			WatchPaths:    ptr([]string{"src/**"}),
			AutoDeploy:    ptr(false),
		})

		got, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.True(t, got.Dockerfile.Valid)
		require.Equal(t, "Dockerfile.prod", got.Dockerfile.String)
		require.Equal(t, "./app", got.DockerContext)
		require.Equal(t, []string{"src/**"}, []string(got.WatchPaths))
		require.False(t, got.AutoDeploy)
	})

	t.Run("runtime settings with healthcheck defaults", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project:          env.projectID,
			App:              env.appID,
			Environment:      env.environmentID,
			Port:             ptr(9090),
			CpuMillicores:    ptr(2000),
			MemoryMib:        ptr(1024),
			StorageMib:       ptr(2048),
			Command:          ptr([]string{"./server", "--prod"}),
			ShutdownSignal:   ptr(openapi.SIGINT),
			UpstreamProtocol: ptr(openapi.H2c),
			OpenapiSpecPath:  nullable.NewNullableWithValue("/openapi.yaml"),
			Healthcheck: nullable.NewNullableWithValue(openapi.EnvironmentHealthcheck{
				Method:          openapi.GET,
				Path:            "/health",
				IntervalSeconds: ptr(15),
			}),
		})

		got, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		rt := got.AppRuntimeSetting
		require.Equal(t, int32(9090), rt.Port)
		require.Equal(t, int32(2000), rt.CpuMillicores)
		require.Equal(t, int32(1024), rt.MemoryMib)
		require.Equal(t, uint32(2048), rt.StorageMib)
		require.Equal(t, []string{"./server", "--prod"}, []string(rt.Command))
		require.Equal(t, db.AppRuntimeSettingsShutdownSignalSIGINT, rt.ShutdownSignal)
		require.Equal(t, db.AppRuntimeSettingsUpstreamProtocolH2c, rt.UpstreamProtocol)
		require.True(t, rt.OpenapiSpecPath.Valid)
		require.Equal(t, "/openapi.yaml", rt.OpenapiSpecPath.String)

		require.True(t, rt.Healthcheck.Valid)
		require.NotNil(t, rt.Healthcheck.Healthcheck)
		require.Equal(t, "GET", rt.Healthcheck.Healthcheck.Method)
		require.Equal(t, "/health", rt.Healthcheck.Healthcheck.Path)
		require.Equal(t, 15, rt.Healthcheck.Healthcheck.IntervalSeconds)
		require.Equal(t, 5, rt.Healthcheck.Healthcheck.TimeoutSeconds, "default applied")
		require.Equal(t, 3, rt.Healthcheck.Healthcheck.FailureThreshold, "default applied")
		require.Equal(t, 0, rt.Healthcheck.Healthcheck.InitialDelaySeconds, "default applied")

		// sentinel_config must be preserved (never in the UPDATE set list).
		require.Equal(t, []byte("{}"), rt.SentinelConfig)
	})

	t.Run("fully specified healthcheck stores verbatim", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Healthcheck: nullable.NewNullableWithValue(openapi.EnvironmentHealthcheck{
				Method:              openapi.GET,
				Path:                "/v1/liveness",
				IntervalSeconds:     ptr(5),
				TimeoutSeconds:      ptr(5),
				FailureThreshold:    ptr(3),
				InitialDelaySeconds: ptr(0),
			}),
		})

		got, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		hc := got.AppRuntimeSetting.Healthcheck
		require.True(t, hc.Valid)
		require.NotNil(t, hc.Healthcheck)
		require.Equal(t, "GET", hc.Healthcheck.Method)
		require.Equal(t, "/v1/liveness", hc.Healthcheck.Path)
		require.Equal(t, 5, hc.Healthcheck.IntervalSeconds)
		require.Equal(t, 5, hc.Healthcheck.TimeoutSeconds)
		require.Equal(t, 3, hc.Healthcheck.FailureThreshold)
		require.Equal(t, 0, hc.Healthcheck.InitialDelaySeconds)
	})

	t.Run("clear nullable fields", func(t *testing.T) {
		env := seedEnvironment(t, h)
		// Seed has dockerfile set; clearing it must null the column.
		call(t, handler.Request{
			Project:         env.projectID,
			App:             env.appID,
			Environment:     env.environmentID,
			Dockerfile:      nullable.NewNullNullable[string](),
			OpenapiSpecPath: nullable.NewNullNullable[string](),
		})

		build, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.False(t, build.Dockerfile.Valid, "dockerfile should be cleared")

		rt, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.False(t, rt.AppRuntimeSetting.OpenapiSpecPath.Valid, "openapiSpecPath should be cleared")
	})

	t.Run("partial update preserves untouched fields", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project:       env.projectID,
			App:           env.appID,
			Environment:   env.environmentID,
			CpuMillicores: ptr(500),
		})

		rt, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Equal(t, int32(500), rt.AppRuntimeSetting.CpuMillicores)
		require.Equal(t, int32(128), rt.AppRuntimeSetting.MemoryMib, "memory untouched, keeps seed default")
		require.Equal(t, int32(8080), rt.AppRuntimeSetting.Port, "port untouched, keeps seed default")
	})

	t.Run("regions create and update", func(t *testing.T) {
		env := seedEnvironment(t, h)

		// Create: one region with bounds 1..3.
		create := []openapi.EnvironmentRegion{regionSetting("us-east-1", 1, 3)}
		call(t, handler.Request{
			Project: env.projectID, App: env.appID, Environment: env.environmentID,
			Regions: &create,
		})
		rows, err := db.Query.ListAppRegionalSettingsByAppEnv(ctx, h.DB.RO(), db.ListAppRegionalSettingsByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, int32(3), rows[0].Replicas, "replicas tracks max")
		require.True(t, rows[0].HorizontalAutoscalingPolicyID.Valid, "policy attached")
		firstPolicyID := rows[0].HorizontalAutoscalingPolicyID.String

		// Update: same region, new bounds 2..2. Policy id must be reused.
		update := []openapi.EnvironmentRegion{regionSetting("us-east-1", 2, 2)}
		call(t, handler.Request{
			Project: env.projectID, App: env.appID, Environment: env.environmentID,
			Regions: &update,
		})
		rows, err = db.Query.ListAppRegionalSettingsByAppEnv(ctx, h.DB.RO(), db.ListAppRegionalSettingsByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Len(t, rows, 1)
		require.Equal(t, int32(2), rows[0].Replicas)
		require.Equal(t, firstPolicyID, rows[0].HorizontalAutoscalingPolicyID.String, "policy reused on update")
	})

	t.Run("multiple regions share one policy", func(t *testing.T) {
		env := seedEnvironment(t, h)

		regions := []openapi.EnvironmentRegion{
			regionSetting("us-east-1", 1, 3),
			regionSetting("us-west-2", 1, 3),
		}
		call(t, handler.Request{
			Project: env.projectID, App: env.appID, Environment: env.environmentID,
			Regions: &regions,
		})

		rows, err := db.Query.ListAppRegionalSettingsByAppEnv(ctx, h.DB.RO(), db.ListAppRegionalSettingsByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Len(t, rows, 2)
		require.True(t, rows[0].HorizontalAutoscalingPolicyID.Valid)
		require.Equal(t,
			rows[0].HorizontalAutoscalingPolicyID.String,
			rows[1].HorizontalAutoscalingPolicyID.String,
			"all regions in an environment share one autoscaling policy",
		)
		require.Equal(t, int32(3), rows[0].Replicas)
		require.Equal(t, int32(3), rows[1].Replicas)
	})

	t.Run("noop when no fields provided", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project: env.projectID, App: env.appID, Environment: env.environmentID,
		})

		rt, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Equal(t, int32(8080), rt.AppRuntimeSetting.Port, "unchanged")
	})
}
