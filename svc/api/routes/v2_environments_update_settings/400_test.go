package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_environments_update_settings"
)

func TestUpdateSettingsBadRequest(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	seedRegions(t, h, "us-east-1", "us-west-2")

	testCases := []struct {
		name    string
		regions []openapi.RegionSetting
	}{
		{name: "min greater than max", regions: []openapi.RegionSetting{regionSetting("us-east-1", 3, 1)}},
		{name: "max above limit", regions: []openapi.RegionSetting{regionSetting("us-east-1", 1, 5)}},
		{name: "min below one", regions: []openapi.RegionSetting{regionSetting("us-east-1", 0, 2)}},
		{name: "unknown region", regions: []openapi.RegionSetting{regionSetting("ap-south-1", 1, 2)}},
		{name: "duplicate region", regions: []openapi.RegionSetting{regionSetting("us-east-1", 1, 2), regionSetting("us-east-1", 1, 3)}},
		{name: "mismatched replica bounds", regions: []openapi.RegionSetting{regionSetting("us-east-1", 1, 3), regionSetting("us-west-2", 2, 4)}},
		{name: "empty regions list", regions: []openapi.RegionSetting{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, headers, handler.Request{
				Project:     env.projectID,
				App:         env.appID,
				Environment: env.environmentID,
				Regions:     &tc.regions,
			})
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400 for %q, got: %s", tc.name, res.RawBody)
		})
	}
}

// The test seeder creates a quota row whose per-instance columns use the schema
// defaults (cpu 2000, memory 4096, storage 10240), so requests above those are 400.
func TestUpdateSettingsResourceQuotaExceeded(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "cpu over quota", req: handler.Request{CpuMillicores: ptr(5000)}},
		{name: "memory over quota", req: handler.Request{MemoryMib: ptr(8192)}},
		{name: "storage over quota", req: handler.Request{StorageMib: ptr(20480)}},
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

// Runtime resources are constrained by the OpenAPI spec: cpu/memory have a floor
// and all three must align to a step (cpu 250, memory 256, storage 512). The
// validation middleware rejects bad shapes before the handler runs.
func TestUpdateSettingsResourceShapeInvalid(t *testing.T) {
	h := testutil.NewHarness(t)

	route := &handler.Handler{DB: h.DB, Auditlogs: h.Auditlogs}
	h.Register(route)

	env := seedEnvironment(t, h)
	rootKey := h.CreateRootKey(env.workspaceID, "environment.*.update_environment")
	headers := authHeaders(rootKey)

	testCases := []struct {
		name string
		req  handler.Request
	}{
		{name: "cpu below floor", req: handler.Request{CpuMillicores: ptr(100)}},
		{name: "cpu off step", req: handler.Request{CpuMillicores: ptr(1300)}},
		{name: "memory below floor", req: handler.Request{MemoryMib: ptr(128)}},
		{name: "memory off step", req: handler.Request{MemoryMib: ptr(1000)}},
		{name: "storage off step", req: handler.Request{StorageMib: ptr(1000)}},
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
