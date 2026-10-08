package handler_test

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_workspace_get_usage"
)

const gib = 1 << 30

type usageRow struct {
	workspaceID, projectID, appID, environmentID string
	hour                                         time.Time
	cpuSeconds, memoryGiBHours, diskGiBHours     float64
	egressBytes                                  int64
}

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

func headers(rootKey string) http.Header {
	return http.Header{
		"Authorization": {"Bearer " + rootKey},
		"Content-Type":  {"application/json"},
	}
}

func TestGetUsage(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		ProjectName: "Payments",
	})
	workspaceID := setup.Workspace.ID
	rootKey := h.CreateRootKey(workspaceID, fmt.Sprintf("unkey:v1:%s:usage#read", workspaceID))
	projectID := setup.Project.ID
	production := setup.Environment
	preview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		AppID:       setup.App.ID,
		Slug:        "preview",
	})
	worker := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Name:        "KEBAP worker",
		Slug:        "worker",
	})
	workerProduction := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		AppID:       worker.ID,
		Slug:        "production",
	})
	// An app in another project with active keys and no compute, so its
	// project name comes only through the app
	labs := h.CreateProject(seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "Labs",
		Slug:        "labs",
	})
	labsApp := h.CreateApp(seed.CreateAppRequest{
		ID:          uid.New(uid.AppPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   labs.ID,
		Name:        "Edge",
		Slug:        "edge",
	})
	// Ids of a deleted app and environment
	deletedAppID := uid.New(uid.AppPrefix)
	deletedEnvironmentID := uid.New(uid.EnvironmentPrefix)
	// Ids of another workspace's resources must not resolve to their names
	other := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{ProjectName: "Other"})
	otherPreview := h.CreateEnvironment(seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: other.Workspace.ID,
		ProjectID:   other.Project.ID,
		AppID:       other.App.ID,
		Slug:        "preview",
	})

	now := h.Clock.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastMonth := monthStart.AddDate(0, -1, 0)
	beforeLastMonth := lastMonth.Add(-time.Hour)

	insertUsage(t, h, usageRow{workspaceID, projectID, setup.App.ID, production.ID, monthStart, 100.5, 1.5, 0, gib})
	insertUsage(t, h, usageRow{workspaceID, projectID, setup.App.ID, production.ID, monthStart, 50.25, 2.25, 0.5, 0})
	insertUsage(t, h, usageRow{workspaceID, projectID, deletedAppID, deletedEnvironmentID, monthStart, 20, 1, 0, 0})
	// Rows recorded without an app id: preview has one row with its app id, so
	// both rows report as one environment. The worker environment has none, so
	// its app comes from the environment
	insertUsage(t, h, usageRow{workspaceID, projectID, setup.App.ID, preview.ID, monthStart, 10, 0.5, 0, 0})
	insertUsage(t, h, usageRow{workspaceID, projectID, "", preview.ID, monthStart, 5, 0.25, 0, 0})
	insertUsage(t, h, usageRow{workspaceID, projectID, "", workerProduction.ID, monthStart, 2, 0.5, 0, 0})
	insertUsage(t, h, usageRow{workspaceID, other.Project.ID, other.App.ID, other.Environment.ID, monthStart, 1, 0.25, 0, 0})
	insertUsage(t, h, usageRow{workspaceID, other.Project.ID, "", otherPreview.ID, monthStart, 0.5, 0, 0, 0})
	insertUsage(t, h, usageRow{workspaceID, projectID, setup.App.ID, production.ID, lastMonth.Add(time.Hour), 1000, 4, 0, 0})
	insertUsage(t, h, usageRow{workspaceID, projectID, setup.App.ID, production.ID, beforeLastMonth, 5000, 8, 0, 0})

	// A key that goes through two apps counts one time, for the app with more
	// verifications. sharedKey and reverseKey split in opposite directions, so
	// the result does not depend on the order of the random app ids. A key
	// prefers an app id to an empty app id. API verifications are not gateway
	// usage
	sharedKey, reverseKey, deletedAppKey := uid.New(uid.KeyPrefix), uid.New(uid.KeyPrefix), uid.New(uid.KeyPrefix)
	emptyMostKey, noAppKey, otherKey, apiKey := uid.New(uid.KeyPrefix), uid.New(uid.KeyPrefix), uid.New(uid.KeyPrefix), uid.New(uid.KeyPrefix)
	insertVerifications(t, h, workspaceID, sharedKey, setup.App.ID, "gateway", monthStart, 5)
	insertVerifications(t, h, workspaceID, sharedKey, deletedAppID, "gateway", monthStart, 2)
	insertVerifications(t, h, workspaceID, reverseKey, setup.App.ID, "gateway", monthStart, 2)
	insertVerifications(t, h, workspaceID, reverseKey, deletedAppID, "gateway", monthStart, 5)
	insertVerifications(t, h, workspaceID, deletedAppKey, deletedAppID, "gateway", monthStart, 3)
	insertVerifications(t, h, workspaceID, emptyMostKey, "", "gateway", monthStart, 9)
	insertVerifications(t, h, workspaceID, emptyMostKey, labsApp.ID, "gateway", monthStart, 1)
	insertVerifications(t, h, workspaceID, noAppKey, "", "gateway", monthStart, 3)
	insertVerifications(t, h, workspaceID, otherKey, other.App.ID, "gateway", monthStart, 4)
	insertVerifications(t, h, workspaceID, apiKey, setup.App.ID, "api", monthStart, 9)
	labsKey := uid.New(uid.KeyPrefix)
	insertVerifications(t, h, workspaceID, labsKey, labsApp.ID, "gateway", monthStart, 4)

	insertBillable(t, h, "billable_verifications_per_month_v2", workspaceID, monthStart, 40)
	insertBillable(t, h, "billable_ratelimits_per_month_v2", workspaceID, monthStart, 2)
	insertBillable(t, h, "billable_verifications_per_month_v2", workspaceID, lastMonth, 7)
	insertBillable(t, h, "billable_ratelimits_per_month_v2", workspaceID, lastMonth, 1)

	t.Run("current month to date", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(rootKey), handler.Request{})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		projectName := "Payments"
		project := ref(projectID, &projectName)
		appRef := ref(setup.App.ID, &setup.App.Name)
		compute := func(cpu, memory, storage, egress float64) openapi.V2WorkspaceGetUsageCompute {
			return openapi.V2WorkspaceGetUsageCompute{CpuSeconds: cpu, MemoryGiBHours: memory, StorageGiBHours: storage, EgressGiB: egress}
		}
		// Most CPU first
		byEnvironment := []openapi.V2WorkspaceGetUsageByEnvironmentRow{
			{
				Project:     project,
				App:         &appRef,
				Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{Id: production.ID, Slug: &production.Slug},
				Compute:     compute(150.75, 3.75, 0.5, 1),
			},
			{
				Project:     project,
				App:         new(ref(deletedAppID, nil)),
				Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{Id: deletedEnvironmentID, Slug: nil},
				Compute:     compute(20, 1, 0, 0),
			},
			{
				Project:     project,
				App:         &appRef,
				Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{Id: preview.ID, Slug: &preview.Slug},
				Compute:     compute(15, 0.75, 0, 0),
			},
			{
				Project:     project,
				App:         new(ref(worker.ID, &worker.Name)),
				Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{Id: workerProduction.ID, Slug: &workerProduction.Slug},
				Compute:     compute(2, 0.5, 0, 0),
			},
			{
				Project:     ref(other.Project.ID, nil),
				App:         new(ref(other.App.ID, nil)),
				Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{Id: other.Environment.ID, Slug: nil},
				Compute:     compute(1, 0.25, 0, 0),
			},
			{
				Project:     ref(other.Project.ID, nil),
				App:         nil,
				Environment: openapi.V2WorkspaceGetUsageEnvironmentResource{Id: otherPreview.ID, Slug: nil},
				Compute:     compute(0.5, 0, 0, 0),
			},
		}
		labsName := "Labs"
		gateway := func(activeKeys int64) openapi.V2WorkspaceGetUsageGateway {
			return openapi.V2WorkspaceGetUsageGateway{ActiveKeys: activeKeys}
		}
		byApp := []openapi.V2WorkspaceGetUsageByAppRow{
			{Project: &project, App: &appRef, Gateway: gateway(1)},
			{Project: nil, App: new(ref(deletedAppID, nil)), Gateway: gateway(2)},
			{Project: new(ref(labs.ID, &labsName)), App: new(ref(labsApp.ID, &labsApp.Name)), Gateway: gateway(2)},
			{Project: nil, App: nil, Gateway: gateway(1)},
			{Project: nil, App: new(ref(other.App.ID, nil)), Gateway: gateway(1)},
		}
		// Most keys first, then app id order, where a row without an app sorts as an empty id
		appID := func(row openapi.V2WorkspaceGetUsageByAppRow) string {
			if row.App == nil {
				return ""
			}
			return row.App.Id
		}
		slices.SortFunc(byApp, func(a, b openapi.V2WorkspaceGetUsageByAppRow) int {
			if a.Gateway.ActiveKeys != b.Gateway.ActiveKeys {
				return int(b.Gateway.ActiveKeys - a.Gateway.ActiveKeys)
			}
			return strings.Compare(appID(a), appID(b))
		})

		require.Equal(t, openapi.V2WorkspaceGetUsageResponseData{
			Period: openapi.V2WorkspaceGetUsagePeriod{Start: monthStart.UnixMilli(), End: now.UnixMilli()},
			Totals: openapi.V2WorkspaceGetUsageTotals{
				// billable_verifications_per_month_mv_v2 also counts the 9 API verifications of apiKey
				Api:     openapi.V2WorkspaceGetUsageApi{Verifications: 40 + 9, Ratelimits: 2},
				Compute: compute(189.25, 6.25, 0.5, 1),
				Gateway: gateway(7),
			},
			Breakdowns: openapi.V2WorkspaceGetUsageBreakdowns{
				ByEnvironment: byEnvironment,
				ByApp:         byApp,
			},
		}, res.Body.Data)
	})

	t.Run("past month", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(rootKey), handler.Request{Period: &openapi.V2WorkspaceGetUsageRequestPeriod{Year: lastMonth.Year(), Month: int(lastMonth.Month())}})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		data := res.Body.Data
		require.Equal(t, openapi.V2WorkspaceGetUsagePeriod{Start: lastMonth.UnixMilli(), End: monthStart.UnixMilli()}, data.Period)
		require.Equal(t, openapi.V2WorkspaceGetUsageApi{Verifications: 7, Ratelimits: 1}, data.Totals.Api)
		require.Len(t, data.Breakdowns.ByEnvironment, 1)
		require.Equal(t, production.ID, data.Breakdowns.ByEnvironment[0].Environment.Id)
		require.Equal(t, 1000.0, data.Totals.Compute.CpuSeconds)
		require.Empty(t, data.Breakdowns.ByApp)
	})
}

func TestGetUsagePeriod(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:usage#read", workspace.ID))

	for name, tc := range map[string]struct {
		now    time.Time
		period openapi.V2WorkspaceGetUsageRequestPeriod
		want   openapi.V2WorkspaceGetUsagePeriod
	}{
		"current month ends now": {
			now:    time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC),
			period: openapi.V2WorkspaceGetUsageRequestPeriod{Year: 2026, Month: 10},
			want: openapi.V2WorkspaceGetUsagePeriod{
				Start: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
				End:   time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC).UnixMilli(),
			},
		},
		// August starts exactly 90 days before 2026-10-30T00:00Z
		"earliest month at the retention edge": {
			now:    time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC),
			period: openapi.V2WorkspaceGetUsageRequestPeriod{Year: 2026, Month: 8},
			want: openapi.V2WorkspaceGetUsagePeriod{
				Start: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
				End:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC).UnixMilli(),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			h.Clock.Set(tc.now)
			res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(rootKey), handler.Request{Period: &tc.period})
			require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
			require.Equal(t, tc.want, res.Body.Data.Period)
		})
	}
}

func TestGetUsageEmptyMonth(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, fmt.Sprintf("unkey:v1:%s:usage#read", workspace.ID))

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(rootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, openapi.V2WorkspaceGetUsageApi{Verifications: 0, Ratelimits: 0}, res.Body.Data.Totals.Api)
	require.Contains(t, res.RawBody, `"byEnvironment":[]`)
	require.Contains(t, res.RawBody, `"byApp":[]`)
	require.Zero(t, res.Body.Data.Totals.Compute.CpuSeconds)
	require.Zero(t, res.Body.Data.Totals.Gateway.ActiveKeys)
}

func ref(id string, name *string) openapi.V2WorkspaceGetUsageResource {
	return openapi.V2WorkspaceGetUsageResource{Id: id, Name: name}
}

// Each row gets its own container so FINAL does not collapse rows that share
// an hour
func insertUsage(t *testing.T, h *testutil.Harness, row usageRow) {
	t.Helper()
	err := h.ClickHouse.Exec(t.Context(),
		"INSERT INTO default.instance_usage_per_hour_v1 (time, workspace_id, project_id, app_id, environment_id, resource_type, resource_id, container_uid, instance_id, cpu_seconds, memory_gib_hours, disk_gib_hours, network_egress_public_bytes) VALUES (?, ?, ?, ?, ?, 'deployment', ?, ?, ?, ?, ?, ?, ?)",
		row.hour, row.workspaceID, row.projectID, row.appID, row.environmentID,
		uid.New(uid.DeploymentPrefix), uid.New("ctr"), uid.New(uid.InstancePrefix),
		row.cpuSeconds, row.memoryGiBHours, row.diskGiBHours, row.egressBytes,
	)
	require.NoError(t, err)
}

func insertVerifications(t *testing.T, h *testutil.Harness, workspaceID, keyID, appID, source string, month time.Time, count int64) {
	t.Helper()
	err := h.ClickHouse.Exec(t.Context(),
		"INSERT INTO default.key_verifications_per_month_v3 (time, workspace_id, key_space_id, identity_id, external_id, key_id, outcome, source, app_id, tags, count) VALUES (?, ?, ?, '', '', ?, 'VALID', ?, ?, [], ?)",
		month, workspaceID, uid.New(uid.KeySpacePrefix), keyID, source, appID, count,
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
