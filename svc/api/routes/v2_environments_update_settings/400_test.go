package handler_test

import (
	"net/http"
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_update_settings"
)

// All bad input returns 400, whether rejected by the OpenAPI validation
// middleware (shapes, patterns, bounds, array caps) or by the handler (quota,
// region logic). The seeded quota row uses the schema defaults (cpu 2000, memory
// 4096, storage 10240), so requests above those exceed quota.
func TestUpdateSettings400(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	seedRegions(t, h, "us-east-1", "us-west-2")

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
		{name: "cpu over quota", req: handler.Request{CpuMillicores: ptr(5000)}},
		{name: "memory over quota", req: handler.Request{MemoryMib: ptr(8192)}},
		{name: "storage over quota", req: handler.Request{StorageMib: ptr(20480)}},

		// Resource shape: floor and step (spec).
		{name: "cpu below floor", req: handler.Request{CpuMillicores: ptr(100)}},
		{name: "cpu off step", req: handler.Request{CpuMillicores: ptr(1300)}},
		{name: "memory below floor", req: handler.Request{MemoryMib: ptr(128)}},
		{name: "memory off step", req: handler.Request{MemoryMib: ptr(1000)}},
		{name: "storage off step", req: handler.Request{StorageMib: ptr(1000)}},

		// Path patterns (spec).
		{name: "dockerfile shell chars", req: handler.Request{Dockerfile: nullable.NewNullableWithValue("Dockerfile; rm -rf /")}},
		{name: "dockerContext space", req: handler.Request{DockerContext: ptr("./my app")}},
		{name: "openapiSpecPath space", req: handler.Request{OpenapiSpecPath: nullable.NewNullableWithValue("./open api.yaml")}},
		{name: "healthcheck path no slash", req: handler.Request{Healthcheck: nullable.NewNullableWithValue(openapi.Healthcheck{Method: "GET", Path: "health"})}},
		{name: "healthcheck path bad chars", req: handler.Request{Healthcheck: nullable.NewNullableWithValue(openapi.Healthcheck{Method: "GET", Path: "/health check"})}},

		// Array caps (spec).
		{name: "watchPaths over limit", req: handler.Request{WatchPaths: ptr(overLimit(11))}},
		{name: "command over limit", req: handler.Request{Command: ptr(overLimit(11))}},
		{name: "regions over limit", req: handler.Request{Regions: ptr([]openapi.RegionSetting{
			regionSetting("r1", 1, 2), regionSetting("r2", 1, 2), regionSetting("r3", 1, 2),
			regionSetting("r4", 1, 2), regionSetting("r5", 1, 2), regionSetting("r6", 1, 2),
		})}},

		// Region replica bounds (spec).
		{name: "replicas max above limit", req: handler.Request{Regions: ptr([]openapi.RegionSetting{regionSetting("us-east-1", 1, 5)})}},
		{name: "replicas min below one", req: handler.Request{Regions: ptr([]openapi.RegionSetting{regionSetting("us-east-1", 0, 2)})}},
		{name: "empty regions list", req: handler.Request{Regions: ptr([]openapi.RegionSetting{})}},

		// Region logic (handler).
		{name: "replicas min greater than max", req: handler.Request{Regions: ptr([]openapi.RegionSetting{regionSetting("us-east-1", 3, 1)})}},
		{name: "unknown region", req: handler.Request{Regions: ptr([]openapi.RegionSetting{regionSetting("ap-south-1", 1, 2)})}},
		{name: "duplicate region", req: handler.Request{Regions: ptr([]openapi.RegionSetting{regionSetting("us-east-1", 1, 2), regionSetting("us-east-1", 1, 3)})}},
		{name: "mismatched replica bounds", req: handler.Request{Regions: ptr([]openapi.RegionSetting{regionSetting("us-east-1", 1, 3), regionSetting("us-west-2", 2, 4)})}},
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
}
