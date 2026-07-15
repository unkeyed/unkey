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

	t.Run("build command set then clear", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project:      env.projectID,
			App:          env.appID,
			Environment:  env.environmentID,
			BuildCommand: nullable.NewNullableWithValue("pnpm --filter api build"),
		})

		got, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.True(t, got.BuildCommand.Valid)
		require.Equal(t, "pnpm --filter api build", got.BuildCommand.String)

		call(t, handler.Request{
			Project:      env.projectID,
			App:          env.appID,
			Environment:  env.environmentID,
			BuildCommand: nullable.NewNullNullable[string](),
		})

		got, err = db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.False(t, got.BuildCommand.Valid, "build command should be cleared")
	})

	t.Run("watchPaths omit preserves, empty clears", func(t *testing.T) {
		env := seedEnvironment(t, h)

		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			WatchPaths:  ptr([]string{"src/**", "lib/**"}),
		})
		got, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"src/**", "lib/**"}, []string(got.WatchPaths))

		// Omitting watchPaths must leave the stored list untouched.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			AutoDeploy:  ptr(false),
		})
		got, err = db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"src/**", "lib/**"}, []string(got.WatchPaths), "omitted watchPaths must be preserved")

		// An empty array is a meaningful value that clears the list.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			WatchPaths:  ptr([]string{}),
		})
		got, err = db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Empty(t, []string(got.WatchPaths), "empty watchPaths must clear the list")
	})

	t.Run("command omit preserves, empty clears", func(t *testing.T) {
		env := seedEnvironment(t, h)

		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Command:     ptr([]string{"./server", "--prod"}),
		})
		rt, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"./server", "--prod"}, []string(rt.AppRuntimeSetting.Command))

		// Omit command, touch another runtime field: command must be preserved.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Port:        ptr(9090),
		})
		rt, err = db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"./server", "--prod"}, []string(rt.AppRuntimeSetting.Command), "omitted command must be preserved")

		// An empty array is a meaningful value that clears the command.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Command:     ptr([]string{}),
		})
		rt, err = db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.Empty(t, []string(rt.AppRuntimeSetting.Command), "empty command must clear the list")
	})

	t.Run("healthcheck partial sets defaults, omit preserves, null removes", func(t *testing.T) {
		env := seedEnvironment(t, h)

		// Only the required method and path; the optional fields take defaults.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Healthcheck: nullable.NewNullableWithValue(openapi.EnvironmentHealthcheck{
				Method: openapi.GET,
				Path:   "/health",
			}),
		})
		rt, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.True(t, rt.AppRuntimeSetting.Healthcheck.Valid)
		hc := rt.AppRuntimeSetting.Healthcheck.Healthcheck
		require.NotNil(t, hc)
		require.Equal(t, "GET", hc.Method)
		require.Equal(t, "/health", hc.Path)
		require.Equal(t, 10, hc.IntervalSeconds, "default applied")
		require.Equal(t, 5, hc.TimeoutSeconds, "default applied")
		require.Equal(t, 3, hc.FailureThreshold, "default applied")
		require.Equal(t, 0, hc.InitialDelaySeconds, "default applied")

		// Omit healthcheck, touch another runtime field: healthcheck must be preserved.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Port:        ptr(9090),
		})
		rt, err = db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.True(t, rt.AppRuntimeSetting.Healthcheck.Valid, "omitted healthcheck must be preserved")
		require.NotNil(t, rt.AppRuntimeSetting.Healthcheck.Healthcheck)
		require.Equal(t, "/health", rt.AppRuntimeSetting.Healthcheck.Healthcheck.Path)

		// Null removes the healthcheck.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			Healthcheck: nullable.NewNullNullable[openapi.EnvironmentHealthcheck](),
		})
		rt, err = db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.False(t, rt.AppRuntimeSetting.Healthcheck.Valid, "null healthcheck must remove it")
	})

	t.Run("nullable string fields omit preserves", func(t *testing.T) {
		env := seedEnvironment(t, h)

		call(t, handler.Request{
			Project:         env.projectID,
			App:             env.appID,
			Environment:     env.environmentID,
			Dockerfile:      nullable.NewNullableWithValue("Dockerfile.prod"),
			BuildCommand:    nullable.NewNullableWithValue("pnpm --filter api build"),
			OpenapiSpecPath: nullable.NewNullableWithValue("/openapi.yaml"),
		})

		// Touch one unrelated field in each settings group; the nullable fields
		// above are omitted and must survive the partial update.
		call(t, handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			AutoDeploy:  ptr(true),
			Port:        ptr(9090),
		})

		build, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.True(t, build.Dockerfile.Valid, "omitted dockerfile must be preserved")
		require.Equal(t, "Dockerfile.prod", build.Dockerfile.String)
		require.True(t, build.BuildCommand.Valid, "omitted buildCommand must be preserved")
		require.Equal(t, "pnpm --filter api build", build.BuildCommand.String)

		rt, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID: env.appID, EnvironmentID: env.environmentID,
		})
		require.NoError(t, err)
		require.True(t, rt.AppRuntimeSetting.OpenapiSpecPath.Valid, "omitted openapiSpecPath must be preserved")
		require.Equal(t, "/openapi.yaml", rt.AppRuntimeSetting.OpenapiSpecPath.String)
	})

	t.Run("runtime settings with healthcheck defaults", func(t *testing.T) {
		env := seedEnvironment(t, h)
		call(t, handler.Request{
			Project:          env.projectID,
			App:              env.appID,
			Environment:      env.environmentID,
			Port:             ptr(9090),
			VCpus:            ptr(2.0),
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
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			VCpus:       ptr(0.5),
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
