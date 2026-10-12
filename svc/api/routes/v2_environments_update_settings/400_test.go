package handler_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_update_settings"
)

// All bad input returns 400, whether rejected by the OpenAPI validation
// middleware (shapes, patterns, bounds, array caps) or by the handler (limits,
// region logic). The seeded limits row uses the test defaults (cpu 2000, memory
// 4096, storage 10240), so requests above those exceed limits.
func TestUpdateSettings400(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	primaryRegion, secondaryRegion, unschedulableRegion := uid.New("region"), uid.New("region"), uid.New("region")
	seedRegions(t, h, primaryRegion, secondaryRegion)
	seedUnschedulableRegion(t, h, unschedulableRegion)

	overLimit := func(n int) []string {
		s := make([]string, n)
		for i := range s {
			s[i] = "x"
		}
		return s
	}

	testCases := []struct {
		name string
		req  handler.Request
	}{
		// Resource quota (handler).
		{name: "cpu over quota", req: handler.Request{VCpus: new(5.0)}},
		{name: "memory over quota", req: handler.Request{MemoryMib: new(8192)}},
		{name: "storage over quota", req: handler.Request{StorageMib: new(20480)}},

		// Resource shape: floor and step (spec).
		{name: "cpu below floor", req: handler.Request{VCpus: new(0.1)}},
		{name: "cpu off step", req: handler.Request{VCpus: new(1.3)}},
		{name: "memory below floor", req: handler.Request{MemoryMib: new(128)}},
		{name: "memory off step", req: handler.Request{MemoryMib: new(1000)}},
		{name: "storage off step", req: handler.Request{StorageMib: new(1000)}},

		// Path validation. Dockerfile is constrained by the spec; rootDirectory is
		// additionally checked by the handler against the control-plane contract.
		{name: "dockerfile empty", req: handler.Request{Dockerfile: nullable.NewNullableWithValue("")}},
		{name: "rootDirectory empty", req: handler.Request{RootDirectory: new("")}},
		{name: "rootDirectory absolute", req: handler.Request{RootDirectory: new("/api")}},
		{name: "rootDirectory dot prefix", req: handler.Request{RootDirectory: new("./api")}},
		{name: "rootDirectory traversal", req: handler.Request{RootDirectory: new("services/../api")}},
		{name: "rootDirectory fragment", req: handler.Request{RootDirectory: new("services/api#main")}},
		{name: "buildCommand empty", req: handler.Request{BuildCommand: nullable.NewNullableWithValue("")}},
		{name: "buildCommand over maxLength", req: handler.Request{BuildCommand: nullable.NewNullableWithValue(strings.Repeat("x", 1001))}},
		{name: "openapiSpecPath no slash", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue("openapi.yaml")}},
		{name: "openapiSpecPath space", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue("/open api.yaml")}},
		{name: "openapiSpecPath traversal", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue("/../openapi.yaml")}},
		{name: "openapiSpecPath extension only", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue(".yaml")}},
		{name: "openapiSpecPath bare slash", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue("/")}},
		{name: "openapiSpecPath trailing slash", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue("/specs/")}},
		{name: "healthcheck path no slash", req: handler.Request{Healthcheck: nullable.NewNullableWithValue(openapi.EnvironmentHealthcheck{Method: "GET", Path: "health"})}},
		{name: "healthcheck path bad chars", req: handler.Request{Healthcheck: nullable.NewNullableWithValue(openapi.EnvironmentHealthcheck{Method: "GET", Path: "/health check"})}},
		{name: "healthcheck path traversal", req: handler.Request{Healthcheck: nullable.NewNullableWithValue(openapi.EnvironmentHealthcheck{Method: "GET", Path: "/../etc/passwd"})}},

		// Watch path glob validation (handler).
		{name: "watchPaths invalid glob", req: handler.Request{WatchPaths: new([]string{"src/["})}},
		{name: "watchPaths invalid among valid", req: handler.Request{WatchPaths: new([]string{"src/**", "{src,lib"})}},

		// Array caps (spec).
		{name: "watchPaths over limit", req: handler.Request{WatchPaths: new(overLimit(11))}},
		{name: "command over limit", req: handler.Request{Command: new(overLimit(11))}},
		{name: "regions over limit", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{
			regionSetting("r1", 1, 2), regionSetting("r2", 1, 2), regionSetting("r3", 1, 2),
			regionSetting("r4", 1, 2), regionSetting("r5", 1, 2), regionSetting("r6", 1, 2),
		})}},

		// Region replica bounds (handler).
		{name: "replicas max above limit", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting(primaryRegion, 1, 5)})}},
		{name: "replicas min below one", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting(primaryRegion, 0, 2)})}},
		{name: "empty regions list", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{})}},

		// Region logic (handler).
		{name: "replicas min greater than max", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting(primaryRegion, 3, 1)})}},
		{name: "unknown region", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting("ap-south-1", 1, 2)})}},
		{name: "unschedulable region", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting(unschedulableRegion, 1, 2)})}},
		{name: "duplicate region", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting(primaryRegion, 1, 2), regionSetting(primaryRegion, 1, 3)})}},
		{name: "mismatched replica bounds", req: handler.Request{Regions: new([]openapi.EnvironmentRegion{regionSetting(primaryRegion, 1, 3), regionSetting(secondaryRegion, 2, 4)})}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			req.Project = env.projectID
			req.App = env.appID
			req.Environment = env.environmentID

			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 for %q, got: %s", tc.name, res.RawBody)
		})
	}

	t.Run("watchPaths error names the offending pattern", func(t *testing.T) {
		req := handler.Request{
			Project:     env.projectID,
			App:         env.appID,
			Environment: env.environmentID,
			WatchPaths:  new([]string{"src/["}),
		}

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, req)
		require.Equal(t, http.StatusBadRequest, res.Status, "raw body: %s", res.RawBody)
		require.Contains(t, res.Body.Error.Detail, "src/[")
	})
}

func TestUpdateSettingsRejectsAmbiguousRegionNames(t *testing.T) {
	h := testutil.NewHarness(t)
	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs, LimitsCache: h.Caches.WorkspaceLimits}
	h.Register(route)
	ctx := context.Background()

	for _, canSchedule := range []bool{true, false} {
		t.Run(fmt.Sprintf("dev region schedulable=%t", canSchedule), func(t *testing.T) {
			env := seedEnvironment(t, h)
			regionName := uid.New("local")
			seedRegions(t, h, regionName)
			require.NoError(t, db.Query.UpsertRegionWithCanSchedule(ctx, h.DB.RW(), db.UpsertRegionWithCanScheduleParams{
				ID:          uid.New(uid.RegionPrefix),
				Name:        regionName,
				Platform:    "dev",
				CanSchedule: canSchedule,
			}))

			rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(rootKey), handler.Request{
				Project: env.projectID, App: env.appID, Environment: env.environmentID,
				Port:    new(9090),
				Regions: new([]openapi.EnvironmentRegion{regionSetting(regionName, 1, 1)}),
			})
			require.Equal(t, http.StatusBadRequest, res.Status, "raw body: %s", res.RawBody)
			require.Equal(t, fmt.Sprintf("Region '%s' exists on multiple platforms and cannot be selected by name.", regionName), res.Body.Error.Detail)

			rows, err := db.Query.ListAppRegionalSettingsByAppEnv(ctx, h.DB.RO(), db.ListAppRegionalSettingsByAppEnvParams{
				AppID: env.appID, EnvironmentID: env.environmentID,
			})
			require.NoError(t, err)
			require.Empty(t, rows)
			runtime, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
				AppID: env.appID, EnvironmentID: env.environmentID,
			})
			require.NoError(t, err)
			require.Equal(t, int32(8080), runtime.Port)
		})
	}
}
