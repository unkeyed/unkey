package handler_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/oapi-codegen/nullable"
	"github.com/stretchr/testify/require"
	logdrainv1 "github.com/unkeyed/unkey/gen/proto/logdrain/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_limits"
	"google.golang.org/protobuf/proto"
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

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{})
	workspaceID := setup.Workspace.ID
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:limits#read", workspaceID))

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
	require.NoError(t, db.Query.UpdateLogdrainsMax(ctx, h.DB.RW(), db.UpdateLogdrainsMaxParams{
		LogdrainsMax: 3,
		WorkspaceID:  workspaceID,
	}))

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
	insertRow(t, h, schema.BillableVerificationsPerMonthV2{Year: int16(now.Year()), Month: int8(now.Month()), WorkspaceID: workspaceID, Count: 40_000})
	insertRow(t, h, schema.BillableRatelimitsPerMonthV2{Year: int16(now.Year()), Month: int8(now.Month()), WorkspaceID: workspaceID, Count: 2_000})
	insertRow(t, h, schema.BillableVerificationsPerMonthV2{Year: int16(lastMonth.Year()), Month: int8(lastMonth.Month()), WorkspaceID: workspaceID, Count: 999})

	// Usage of another workspace must not count toward this one
	other := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{})
	otherDeployment := createDeployment(t, h, other, 2000, 4096, 4096)
	insertTopology(t, h, other.Workspace.ID, otherDeployment.ID, 5, db.DeploymentTopologyDesiredStatusRunning)
	h.CreateCustomDomain(seed.CreateCustomDomainRequest{
		ID:            uid.New(uid.DomainPrefix),
		WorkspaceID:   other.Workspace.ID,
		ProjectID:     other.Project.ID,
		AppID:         other.App.ID,
		EnvironmentID: other.Environment.ID,
		Domain:        uid.DNS1035() + ".example.com",
	})
	insertLogdrain(t, h, other.Workspace.ID)
	insertRow(t, h, schema.BillableVerificationsPerMonthV2{Year: int16(now.Year()), Month: int8(now.Month()), WorkspaceID: other.Workspace.ID, Count: 7_000})
	insertRow(t, h, schema.BillableRatelimitsPerMonthV2{Year: int16(now.Year()), Month: int8(now.Month()), WorkspaceID: other.Workspace.ID, Count: 7_000})

	res := callGetLimits(h, route, bearer(rootKey))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	require.Equal(t, openapi.V2WorkspaceGetLimitsResponseData{
		ApiBillableOperationsCountMaxPerMonth: meteredLimit(150_000, 42_000),
		ApiRequestsCountMaxPerMinute:          limit(1000),
		LogsRetentionDaysMax:                  limit(7),
		LogsAuditRetentionDaysMax:             limit(30),
		LogdrainsMax:                          meteredLimit(3, 1),
		CpuCoresMax:                           new(meteredLimit(4, 1.5)),
		CpuCoresMaxPerInstance:                new(limit(2)),
		MemoryMibMax:                          new(meteredLimit(8192, 1536)),
		MemoryMibMaxPerInstance:               new(limit(4096)),
		StorageMibMax:                         new(meteredLimit(10_240, 3072)),
		StorageMibMaxPerInstance:              new(limit(5120)),
		BuildsConcurrentMax:                   new(limit(2)),
		AutoscalingReplicasMax:                new(limit(10)),
		CustomDomainsMax:                      new(meteredLimit(5, 1)),
	}, res.Body.Data)
}

func TestGetLimitsWithoutComputePlanOrRequestsPerMinute(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:limits#read", workspace.ID))

	res := callGetLimits(h, route, bearer(rootKey))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	var body struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.RawBody), &body))
	require.ElementsMatch(t, []string{
		"apiBillableOperationsCountMaxPerMonth",
		"apiRequestsCountMaxPerMinute",
		"logsRetentionDaysMax",
		"logsAuditRetentionDaysMax",
		"logdrainsMax",
	}, slices.Collect(maps.Keys(body.Data)))
	var requestsPerMinute map[string]any
	require.NoError(t, json.Unmarshal(body.Data["apiRequestsCountMaxPerMinute"], &requestsPerMinute))
	require.Equal(t, map[string]any{"limit": nil}, requestsPerMinute)
	require.Equal(t, meteredLimit(1_000_000, 0), res.Body.Data.ApiBillableOperationsCountMaxPerMonth)
}

func createDeployment(t *testing.T, h *testutil.Harness, setup testutil.DeploymentTestSetup, cpuMillicores, memoryMib int32, storageMib uint32) db.Deployment {
	t.Helper()
	return h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
		CpuMillicores: cpuMillicores,
		MemoryMib:     memoryMib,
		StorageMib:    storageMib,
	})
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
	config, err := proto.Marshal(&logdrainv1.Config{
		Stream: &logdrainv1.Config_AuditLogs{AuditLogs: &logdrainv1.AuditLogStreamConfig{}},
	})
	require.NoError(t, err)
	require.NoError(t, db.Query.InsertLogdrain(t.Context(), h.DB.RW(), db.InsertLogdrainParams{
		ID:          uid.New(uid.LogdrainPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Stream:      db.LogdrainsStreamAuditLogs,
		Config:      config,
		CreatedAt:   time.Now().UnixMilli(),
	}))
}

func insertRow[T schema.Row](t *testing.T, h *testutil.Harness, row T) {
	t.Helper()
	batch, err := h.ClickHouse.Conn().PrepareBatch(t.Context(), clickhouse.InsertQuery[T]())
	require.NoError(t, err)
	require.NoError(t, batch.AppendStruct(&row))
	require.NoError(t, batch.Send())
}

func TestGetLimitsReturnsNullForUnlimitedCustomDomains(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	// The seeded limits row allows 1,000,000 custom domains, which plans use for unlimited
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{})
	rootKey := h.CreateRootKey(setup.Workspace.ID, fmt.Sprintf("unkey:v1:%s:limits#read", setup.Workspace.ID))

	res := callGetLimits(h, route, bearer(rootKey))
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

	var body struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.RawBody), &body))
	var customDomains map[string]any
	require.NoError(t, json.Unmarshal(body.Data["customDomainsMax"], &customDomains))
	require.Equal(t, map[string]any{"limit": nil, "current": float64(0)}, customDomains)
}

func limit(maximum int64) openapi.V2WorkspaceGetLimitsLimit {
	return openapi.V2WorkspaceGetLimitsLimit{
		Limit:   nullable.NewNullableWithValue(maximum),
		Current: nil,
	}
}

func meteredLimit(maximum int64, current float64) openapi.V2WorkspaceGetLimitsLimit {
	return openapi.V2WorkspaceGetLimitsLimit{
		Limit:   nullable.NewNullableWithValue(maximum),
		Current: &current,
	}
}
