package handler_test

import (
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
		Permissions: []string{"workspace.*.read_usage"},
	})
	workspaceID := setup.Workspace.ID
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
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(setup.RootKey), handler.Request{})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		projectName := "Payments"
		// Most CPU first
		environments := []openapi.V2WorkspaceGetUsageEnvironment{
			{
				ProjectId: projectID, ProjectName: &projectName,
				AppId: setup.App.ID, AppName: &setup.App.Name,
				EnvironmentId: production.ID, EnvironmentSlug: &production.Slug,
				CpuSeconds: 150.75, MemoryGiBHours: 3.75, DiskGiBHours: 0.5, EgressGiB: 1,
			},
			{
				ProjectId: projectID, ProjectName: &projectName,
				AppId: deletedAppID, AppName: nil,
				EnvironmentId: deletedEnvironmentID, EnvironmentSlug: nil,
				CpuSeconds: 20, MemoryGiBHours: 1, DiskGiBHours: 0, EgressGiB: 0,
			},
			{
				ProjectId: projectID, ProjectName: &projectName,
				AppId: setup.App.ID, AppName: &setup.App.Name,
				EnvironmentId: preview.ID, EnvironmentSlug: &preview.Slug,
				CpuSeconds: 15, MemoryGiBHours: 0.75, DiskGiBHours: 0, EgressGiB: 0,
			},
			{
				ProjectId: projectID, ProjectName: &projectName,
				AppId: worker.ID, AppName: &worker.Name,
				EnvironmentId: workerProduction.ID, EnvironmentSlug: &workerProduction.Slug,
				CpuSeconds: 2, MemoryGiBHours: 0.5, DiskGiBHours: 0, EgressGiB: 0,
			},
			{
				ProjectId: other.Project.ID, ProjectName: nil,
				AppId: other.App.ID, AppName: nil,
				EnvironmentId: other.Environment.ID, EnvironmentSlug: nil,
				CpuSeconds: 1, MemoryGiBHours: 0.25, DiskGiBHours: 0, EgressGiB: 0,
			},
			{
				ProjectId: other.Project.ID, ProjectName: nil,
				AppId: "", AppName: nil,
				EnvironmentId: otherPreview.ID, EnvironmentSlug: nil,
				CpuSeconds: 0.5, MemoryGiBHours: 0, DiskGiBHours: 0, EgressGiB: 0,
			},
		}
		labsName := "Labs"
		apps := []openapi.V2WorkspaceGetUsageApp{
			{AppId: setup.App.ID, AppName: &setup.App.Name, ProjectId: &projectID, ProjectName: &projectName, ActiveKeys: 1},
			{AppId: deletedAppID, AppName: nil, ProjectId: nil, ProjectName: nil, ActiveKeys: 2},
			{AppId: labsApp.ID, AppName: &labsApp.Name, ProjectId: &labs.ID, ProjectName: &labsName, ActiveKeys: 2},
			{AppId: "", AppName: nil, ProjectId: nil, ProjectName: nil, ActiveKeys: 1},
			{AppId: other.App.ID, AppName: nil, ProjectId: nil, ProjectName: nil, ActiveKeys: 1},
		}
		// Most keys first, then app id order
		slices.SortFunc(apps, func(a, b openapi.V2WorkspaceGetUsageApp) int {
			if a.ActiveKeys != b.ActiveKeys {
				return int(b.ActiveKeys - a.ActiveKeys)
			}
			return strings.Compare(a.AppId, b.AppId)
		})

		require.Equal(t, openapi.V2WorkspaceGetUsageResponseData{
			Period: openapi.V2WorkspaceGetUsagePeriod{Start: monthStart.UnixMilli(), End: now.UnixMilli()},
			// billable_verifications_per_month_mv_v2 also counts the 9 API verifications of apiKey
			Api: openapi.V2WorkspaceGetUsageApi{Verifications: 40 + 9, Ratelimits: 2},
			Compute: openapi.V2WorkspaceGetUsageCompute{
				CpuSeconds:     189.25,
				MemoryGiBHours: 6.25,
				DiskGiBHours:   0.5,
				EgressGiB:      1,
				ActiveKeys:     7,
				Environments:   environments,
				Apps:           apps,
			},
		}, res.Body.Data)
	})

	t.Run("previous month", func(t *testing.T) {
		res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(setup.RootKey), handler.Request{Period: new(openapi.UsagePeriodPrevious)})
		require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)

		require.Equal(t, openapi.V2WorkspaceGetUsagePeriod{Start: lastMonth.UnixMilli(), End: monthStart.UnixMilli()}, res.Body.Data.Period)
		require.Equal(t, openapi.V2WorkspaceGetUsageApi{Verifications: 7, Ratelimits: 1}, res.Body.Data.Api)
		require.Len(t, res.Body.Data.Compute.Environments, 1)
		require.Equal(t, production.ID, res.Body.Data.Compute.Environments[0].EnvironmentId)
		require.Equal(t, 1000.0, res.Body.Data.Compute.CpuSeconds)
		require.Empty(t, res.Body.Data.Compute.Apps)
	})
}

func TestGetUsageEmptyMonth(t *testing.T) {
	h := testutil.NewHarness(t, testutil.HarnessConfig{ClickHouse: true})
	route := newRoute(h)
	h.Register(route)

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID, "workspace.*.read_usage")

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, headers(rootKey), handler.Request{})
	require.Equal(t, http.StatusOK, res.Status, "expected 200, received: %s", res.RawBody)
	require.Equal(t, openapi.V2WorkspaceGetUsageApi{Verifications: 0, Ratelimits: 0}, res.Body.Data.Api)
	require.Contains(t, res.RawBody, `"environments":[]`)
	require.Contains(t, res.RawBody, `"apps":[]`)
	require.Zero(t, res.Body.Data.Compute.CpuSeconds)
	require.Zero(t, res.Body.Data.Compute.ActiveKeys)
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
