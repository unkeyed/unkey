package handler_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_limits"
)

func newRoute(h *testutil.Harness) *handler.Handler {
	// The harness leaves ClickHouse nil unless HarnessConfig enables it
	ch := h.ClickHouse
	if ch == nil {
		ch = clickhouse.NewNoop()
	}
	return &handler.Handler{
		DB:         h.DB,
		ClickHouse: ch,
		Clock:      h.Clock,
	}
}

func callGetLimits(h *testutil.Harness, route *handler.Handler, headers http.Header) testutil.TestResponse[handler.Response] {
	req := httptest.NewRequest(route.Method(), route.Path(), nil)
	req.Header = headers
	return testutil.CallRaw[handler.Response](h, req)
}

func bearer(rootKey string) http.Header {
	return http.Header{"Authorization": {"Bearer " + rootKey}}
}

func TestGetLimitsWithComputePlan(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)
	ctx := t.Context()

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"workspace.*.read_limits"},
	})
	workspaceID := setup.Workspace.ID

	err := db.Query.UpsertLimit(ctx, h.DB.RW(), db.UpsertLimitParams{
		WorkspaceID:                           workspaceID,
		ApiBillableOperationsCountMaxPerMonth: 150_000,
		ApiRequestsCountMaxPerMinute:          sql.NullInt32{Int32: 1000, Valid: true},
		LogsRetentionDaysMax:                  7,
		LogsAuditRetentionDaysMax:             30,
		TeamEnabled:                           false,
		CpuCoresMax:                           4,
		CpuCoresMaxPerInstance:                2,
		MemoryMibMax:                          8192,
		MemoryMibMaxPerInstance:               4096,
		StorageMibMax:                         10_240,
		StorageMibMaxPerInstance:              5120,
		BuildsConcurrentMax:                   2,
		CustomDomainsMax:                      5,
		AutoscalingReplicasMax:                10,
	})
	require.NoError(t, err)
	_, err = h.DB.RW().ExecContext(ctx, "UPDATE limits SET logdrains_max = 3 WHERE workspace_id = ?", workspaceID)
	require.NoError(t, err)

	running := createDeployment(t, h, setup, 500, 512, 1024)
	insertTopology(t, h, workspaceID, running.ID, 2, db.DeploymentTopologyDesiredStatusRunning)
	insertTopology(t, h, workspaceID, running.ID, 1, db.DeploymentTopologyDesiredStatusRunning)
	stopped := createDeployment(t, h, setup, 1000, 2048, 4096)
	insertTopology(t, h, workspaceID, stopped.ID, 3, db.DeploymentTopologyDesiredStatusStopped)

	h.CreateCustomDomain(seed.CreateCustomDomainRequest{
		ID:            uid.New(uid.DomainPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		Domain:        uid.DNS1035() + ".example.com",
	})
	insertLogdrain(t, h, workspaceID)

	now := h.Clock.Now().UTC()
	lastMonth := now.AddDate(0, -1, 0)
	insertBillable(t, h, "billable_verifications_per_month_v2", workspaceID, now, 40_000)
	insertBillable(t, h, "billable_ratelimits_per_month_v2", workspaceID, now, 2_000)
	insertBillable(t, h, "billable_verifications_per_month_v2", workspaceID, lastMonth, 999)

	res := callGetLimits(h, route, bearer(setup.RootKey))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	requestsPerMinute := int64(1000)
	require.Equal(t, openapi.V2WorkspaceGetLimitsResponseData{
		Api: openapi.V2WorkspaceGetLimitsApi{
			BillableOperations: openapi.LimitMeter{Limit: 150_000, Used: 42_000},
			RequestsPerMinute:  &requestsPerMinute,
		},
		Logs: openapi.V2WorkspaceGetLimitsLogs{
			RetentionDays:      7,
			AuditRetentionDays: 30,
			LogDrains:          openapi.LimitMeter{Limit: 3, Used: 1},
		},
		Compute: &openapi.V2WorkspaceGetLimitsCompute{
			VCpus:                 openapi.V2WorkspaceGetLimitsVcpuMeter{Limit: 4, Used: 1.5},
			VCpusPerInstance:      2,
			MemoryMib:             openapi.LimitMeter{Limit: 8192, Used: 1536},
			MemoryMibPerInstance:  4096,
			StorageMib:            openapi.LimitMeter{Limit: 10_240, Used: 3072},
			StorageMibPerInstance: 5120,
			ConcurrentBuilds:      2,
			ReplicasPerRegion:     10,
			CustomDomains:         openapi.LimitMeter{Limit: 5, Used: 1},
		},
	}, res.Body.Data)
}

func TestGetLimitsOmitsComputeAndRequestsPerMinute(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "workspace.*.read_limits")

	res := callGetLimits(h, route, bearer(rootKey))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	var body struct {
		Data struct {
			Api map[string]json.RawMessage `json:"api"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.RawBody), &body))
	require.NotContains(t, body.Data.Api, "requestsPerMinute")
	require.NotContains(t, res.RawBody, `"compute"`)
	require.Equal(t, openapi.LimitMeter{Limit: 1_000_000, Used: 0}, res.Body.Data.Api.BillableOperations)
}

func createDeployment(t *testing.T, h *testutil.Harness, setup testutil.DeploymentTestSetup, cpuMillicores, memoryMib, storageMib int) db.Deployment {
	t.Helper()
	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	_, err := h.DB.RW().ExecContext(t.Context(),
		"UPDATE deployments SET cpu_millicores = ?, memory_mib = ?, storage_mib = ? WHERE id = ?",
		cpuMillicores, memoryMib, storageMib, deployment.ID,
	)
	require.NoError(t, err)
	return deployment
}

func insertTopology(t *testing.T, h *testutil.Harness, workspaceID, deploymentID string, replicasMax uint32, status db.DeploymentTopologyDesiredStatus) {
	t.Helper()
	err := db.Query.InsertDeploymentTopology(t.Context(), h.DB.RW(), db.InsertDeploymentTopologyParams{
		WorkspaceID:            workspaceID,
		DeploymentID:           deploymentID,
		RegionID:               uid.New(uid.RegionPrefix),
		AutoscalingReplicasMin: 1,
		AutoscalingReplicasMax: replicasMax,
		DesiredStatus:          status,
		CreatedAt:              time.Now().UnixMilli(),
	})
	require.NoError(t, err)
}

func insertLogdrain(t *testing.T, h *testutil.Harness, workspaceID string) {
	t.Helper()
	_, err := h.DB.RW().ExecContext(t.Context(),
		"INSERT INTO logdrains (id, workspace_id, name, stream, config, lease_id, fencing_token, created_at) VALUES (?, ?, ?, 'audit_logs', ?, '', '', ?)",
		uid.New(uid.LogdrainPrefix), workspaceID, "KEBAP", []byte("{}"), time.Now().UnixMilli(),
	)
	require.NoError(t, err)
}

func insertBillable(t *testing.T, h *testutil.Harness, table, workspaceID string, month time.Time, count int64) {
	t.Helper()
	err := h.ClickHouse.Exec(t.Context(),
		"INSERT INTO default."+table+" (year, month, workspace_id, count) VALUES (?, ?, ?, ?)",
		month.Year(), int(month.Month()), workspaceID, count,
	)
	require.NoError(t, err)
}
