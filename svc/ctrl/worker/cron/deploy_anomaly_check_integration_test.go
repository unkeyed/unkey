package cron_test

import (
	"context"
	"database/sql"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/go-faster/city"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type anomalyAdmins struct {
	mu    sync.RWMutex
	orgID string
}

func (a *anomalyAdmins) AdminEmails(_ context.Context, orgID string) ([]string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if orgID != a.orgID {
		return nil, nil
	}
	return []string{"admin@example.com"}, nil
}

func (a *anomalyAdmins) allow(orgID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.orgID = orgID
}

type anomalyTestApp struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deploymentID  string
	orgID         string
}

// TestRunDeployAnomalyCheck_Integration covers production filtering, alert
// persistence, snapshot-based recovery, and both notification templates.
func TestRunDeployAnomalyCheck_Integration(t *testing.T) {
	sender := email.NewCapture()
	admins := &anomalyAdmins{}
	h := harness.New(t, harness.WithDeployAnomalyNotifications(admins, sender))
	production := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	admins.allow(production.orgID)
	preview := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindPreview)
	windowStart := uniqueAnomalyWindowStart()

	for i := range 12 {
		bucket := windowStart.Add(-time.Duration(12-i) * 5 * time.Minute)
		insertAnomalyRequestBucket(t, h, production, bucket, 1_000)
		insertAnomalyRequestBucket(t, h, preview, bucket, 1_000)
	}
	insertAnomalyRequestBucket(t, h, production, windowStart, 10_000)
	insertAnomalyRequestBucket(t, h, preview, windowStart, 10_000)
	advanceResourceWatermark(t, h, production, windowStart)

	runAnomalyWindow(t, h, windowStart)

	var alert db.AlertEvent
	err := h.DB.RO().QueryRowContext(h.Ctx, `
		SELECT pk, id, workspace_id, project_id, app_id, environment_id,
			deployment_id, metric, status, fired_at, last_seen_at, resolved_at,
			resolved_by, resolution_message, observed_value, baseline_mean,
			baseline_stddev, threshold_sigma, window_start, window_end,
			created_at, updated_at
		FROM alert_events WHERE workspace_id = ?`, production.workspaceID).
		Scan(
			&alert.Pk, &alert.ID, &alert.WorkspaceID, &alert.ProjectID, &alert.AppID,
			&alert.EnvironmentID, &alert.DeploymentID, &alert.Metric, &alert.Status,
			&alert.FiredAt, &alert.LastSeenAt, &alert.ResolvedAt, &alert.ResolvedBy,
			&alert.ResolutionMessage, &alert.ObservedValue, &alert.BaselineMean,
			&alert.BaselineStddev, &alert.ThresholdSigma, &alert.WindowStart,
			&alert.WindowEnd, &alert.CreatedAt, &alert.UpdatedAt,
		)
	require.NoError(t, err)
	require.Equal(t, db.AlertEventsMetricRequests, alert.Metric)
	require.Equal(t, db.AlertEventsStatusOpen, alert.Status)
	require.Equal(t, 10_000.0, alert.ObservedValue)
	require.Equal(t, 1_000.0, alert.BaselineMean)
	require.Equal(t, 100.0, alert.BaselineStddev)
	require.Equal(t, 4.0, alert.ThresholdSigma)
	require.Equal(t, windowStart.UnixMilli(), alert.WindowStart)
	require.Equal(t, windowStart.Add(5*time.Minute).UnixMilli(), alert.WindowEnd)
	var productionAlerts int
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM alert_events WHERE workspace_id = ?", production.workspaceID).
		Scan(&productionAlerts))
	require.Equal(t, 1, productionAlerts)

	var previewAlerts int
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM alert_events WHERE workspace_id = ?", preview.workspaceID).
		Scan(&previewAlerts))
	require.Zero(t, previewAlerts)
	require.Equal(t, 1, sender.CountByTemplate("deploy-anomaly-alert"))

	quietStart := assertIncompleteTelemetryNoop(t, h, production, alert.ID, windowStart)
	for i := 1; i <= 3; i++ {
		quietWindow := quietStart.Add(time.Duration(i) * 5 * time.Minute)
		insertAnomalyRequestBucket(t, h, production, quietWindow, 1_000)
		advanceResourceWatermark(t, h, production, quietWindow)
		runAnomalyWindow(t, h, quietWindow)

		var status string
		require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
			"SELECT status FROM alert_events WHERE id = ?", alert.ID).Scan(&status))
		if i < 3 {
			require.Equal(t, "open", status)
		} else {
			require.Equal(t, "resolved", status)
		}
	}

	var resolvedBy, resolutionMessage sql.NullString
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT resolved_by, resolution_message FROM alert_events WHERE id = ?", alert.ID).
		Scan(&resolvedBy, &resolutionMessage))
	require.Equal(t, "system", resolvedBy.String)
	require.Equal(t, "Metric returned to baseline for 3 consecutive windows", resolutionMessage.String)
	require.Equal(t, 1, sender.CountByTemplate("deploy-anomaly-resolved"))
	assertStoppedDeploymentSuppression(t, h)
}

func assertIncompleteTelemetryNoop(
	t *testing.T,
	h *harness.Harness,
	openApp anomalyTestApp,
	alertID string,
	windowStart time.Time,
) time.Time {
	t.Helper()
	client := hydrav1.NewDeployAnomalyServiceIngressClient(h.Restate,
		openApp.workspaceID+"-"+openApp.appID+"-"+openApp.environmentID)
	incompleteOpenMetric := &hydrav1.DeployAnomalyMetricInput{
		Metric:    string(db.AlertEventsMetricRequests),
		DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE,
	}
	for i := range 4 {
		_, err := client.Evaluate().Request(h.Ctx, &hydrav1.EvaluateDeployAnomalyRequest{
			WindowStart: windowStart.Add(time.Duration(i+1) * 5 * time.Minute).UnixMilli(),
			WindowEnd:   windowStart.Add(time.Duration(i+2) * 5 * time.Minute).UnixMilli(),
			WorkspaceId: openApp.workspaceID, ProjectId: openApp.projectID,
			AppId: openApp.appID, EnvironmentId: openApp.environmentID,
			DeploymentId: openApp.deploymentID, DeploymentDesiredState: "running",
			Metrics: []*hydrav1.DeployAnomalyMetricInput{incompleteOpenMetric},
		})
		require.NoError(t, err)
	}
	var status string
	var lastSeenAt int64
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT status, last_seen_at FROM alert_events WHERE id = ?", alertID).Scan(&status, &lastSeenAt))
	require.Equal(t, "open", status, "incomplete windows must not resolve an open alert")
	require.Equal(t, windowStart.Add(5*time.Minute).UnixMilli(), lastSeenAt, "incomplete windows must not touch an open alert")

	newApp := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	newClient := hydrav1.NewDeployAnomalyServiceIngressClient(h.Restate,
		newApp.workspaceID+"-"+newApp.appID+"-"+newApp.environmentID)
	incomplete := &hydrav1.DeployAnomalyMetricInput{
		Metric:    string(db.AlertEventsMetricRequestsDrop),
		DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE,
		Current:   0, BaselineMean: 1_000, ObservedBaselineBuckets: 288,
		RecentMedianRequests: 1_000, RecentActiveBuckets: 12,
	}
	for i := range 2 {
		_, err := newClient.Evaluate().Request(h.Ctx, &hydrav1.EvaluateDeployAnomalyRequest{
			WindowStart: windowStart.Add(time.Duration(i) * 5 * time.Minute).UnixMilli(),
			WindowEnd:   windowStart.Add(time.Duration(i+1) * 5 * time.Minute).UnixMilli(),
			WorkspaceId: newApp.workspaceID, ProjectId: newApp.projectID,
			AppId: newApp.appID, EnvironmentId: newApp.environmentID,
			DeploymentId: newApp.deploymentID, DeploymentDesiredState: "running",
			Metrics: []*hydrav1.DeployAnomalyMetricInput{incomplete},
		})
		require.NoError(t, err)
	}
	var alerts int
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM alert_events WHERE app_id = ?", newApp.appID).Scan(&alerts))
	require.Zero(t, alerts, "incomplete windows must not open a request-drop alert")
	return windowStart
}

func assertStoppedDeploymentSuppression(t *testing.T, h *harness.Harness) {
	t.Helper()
	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	windowStart := time.Now().UTC().Truncate(5 * time.Minute)
	alertID := uid.New(uid.AlertPrefix)
	require.NoError(t, h.DB.InsertAlertEvent(h.Ctx, db.InsertAlertEventParams{
		ID: alertID, WorkspaceID: app.workspaceID, ProjectID: app.projectID,
		AppID: app.appID, EnvironmentID: app.environmentID,
		DeploymentID: sql.NullString{String: app.deploymentID, Valid: true},
		Metric:       db.AlertEventsMetricRequestsDrop, FiredAt: windowStart.UnixMilli(),
		LastSeenAt: windowStart.UnixMilli(), ObservedValue: 0, BaselineMean: 1_000,
		BaselineStddev: 0, ThresholdSigma: 0, WindowStart: windowStart.UnixMilli(),
		WindowEnd: windowStart.Add(5 * time.Minute).UnixMilli(), CreatedAt: windowStart.UnixMilli(),
		UpdatedAt: sql.NullInt64{},
	}))

	client := hydrav1.NewDeployAnomalyServiceIngressClient(h.Restate,
		app.workspaceID+"-"+app.appID+"-"+app.environmentID)
	request := &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowStart.Add(5 * time.Minute).UnixMilli(),
		WindowEnd:   windowStart.Add(10 * time.Minute).UnixMilli(),
		WorkspaceId: app.workspaceID, ProjectId: app.projectID,
		AppId: app.appID, EnvironmentId: app.environmentID,
		DeploymentId: app.deploymentID, DeploymentDesiredState: "stopped",
		NotificationsMuted: true,
		Metrics: []*hydrav1.DeployAnomalyMetricInput{{
			Metric:    string(db.AlertEventsMetricRequestsDrop),
			DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
			Current:   0, BaselineMean: 1_000, ObservedBaselineBuckets: 288,
			RecentMedianRequests: 1_000, RecentActiveBuckets: 12,
		}},
	}
	_, err := client.Evaluate().Request(h.Ctx, request)
	require.NoError(t, err)

	var status string
	var resolutionMessage sql.NullString
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT status, resolution_message FROM alert_events WHERE id = ?", alertID).
		Scan(&status, &resolutionMessage))
	require.Equal(t, "resolved", status)
	require.Equal(t, "Deployment stopped", resolutionMessage.String)

	request.WindowStart += int64(5 * time.Minute / time.Millisecond)
	request.WindowEnd += int64(5 * time.Minute / time.Millisecond)
	_, err = client.Evaluate().Request(h.Ctx, request)
	require.NoError(t, err)
	var alerts int
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM alert_events WHERE app_id = ?", app.appID).Scan(&alerts))
	require.Equal(t, 1, alerts, "a stopped deployment must not open a replacement request-drop alert")
}

func createAnomalyTestApp(t *testing.T, h *harness.Harness, kind mysqltype.EnvironmentKind) anomalyTestApp {
	t.Helper()
	workspace := h.Seed.CreateWorkspace(h.Ctx)
	project := h.Seed.CreateProject(h.Ctx, seed.CreateProjectRequest{
		ID: uid.New(uid.ProjectPrefix), WorkspaceID: workspace.ID,
		Name: "anomaly-project", Slug: uid.New("slug"), DeleteProtection: false,
	})
	app := h.Seed.CreateApp(h.Ctx, seed.CreateAppRequest{
		ID: uid.New("app"), WorkspaceID: workspace.ID, ProjectID: project.ID,
		Name: "anomaly-app", Slug: uid.New("slug"), DefaultBranch: "main",
	})
	environment := h.Seed.CreateEnvironment(h.Ctx, seed.CreateEnvironmentRequest{
		ID: uid.New("env"), WorkspaceID: workspace.ID, ProjectID: project.ID,
		AppID: app.ID, Slug: string(kind), Kind: kind,
	})
	deployment := h.Seed.CreateDeployment(h.Ctx, seed.CreateDeploymentRequest{
		WorkspaceID: workspace.ID, ProjectID: project.ID, AppID: app.ID,
		EnvironmentID: environment.ID, Status: mysqltype.DeploymentsStatusReady,
	})
	require.NoError(t, h.DB.UpdateAppDeployments(h.Ctx, db.UpdateAppDeploymentsParams{
		CurrentDeploymentID: sql.NullString{String: deployment.ID, Valid: true},
		IsRolledBack:        false, UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true}, AppID: app.ID,
	}))
	return anomalyTestApp{
		workspaceID: workspace.ID, projectID: project.ID, appID: app.ID,
		environmentID: environment.ID, deploymentID: deployment.ID, orgID: workspace.OrgID,
	}
}

func insertAnomalyRequestBucket(t *testing.T, h *harness.Harness, app anomalyTestApp, bucket time.Time, requests int64) {
	t.Helper()
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, `
		INSERT INTO default.frontline_requests_per_5m_v1
			(time, workspace_id, project_id, app_id, environment_id, deployment_id, response_status, count)
		VALUES (?, ?, ?, ?, ?, ?, 200, ?)
	`, bucket, app.workspaceID, app.projectID, app.appID, app.environmentID, app.deploymentID, requests))
}

func advanceResourceWatermark(t *testing.T, h *harness.Harness, app anomalyTestApp, windowStart time.Time) {
	t.Helper()
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, `
		INSERT INTO default.instance_resources_per_minute_v1 (
			time, workspace_id, project_id, app_id, environment_id,
			resource_type, resource_id, container_uid, instance_id,
			cpu_usage_usec_min, cpu_usage_usec_max,
			memory_bytes_max, memory_allocated_bytes_max,
			network_egress_public_bytes_min, network_egress_public_bytes_max
		) VALUES (?, ?, ?, ?, ?, 'deployment', ?, ?, ?, 0, 0, 10, 100, 0, 0)
	`, windowStart.Add(4*time.Minute), app.workspaceID, app.projectID, app.appID,
		app.environmentID, app.deploymentID, "container", "instance"))
}

func runAnomalyWindow(t *testing.T, h *harness.Harness, windowStart time.Time) {
	t.Helper()
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, `
		INSERT INTO default.anomaly_source_watermarks_v1 (source, region, time)
		SELECT
			source,
			region,
			if(source = 'resources', toDateTime(?), toDateTime(?)) AS time
		FROM default.anomaly_source_watermarks_v1
		WHERE anomaly_source_watermarks_v1.time > now() - INTERVAL 2 HOUR
		GROUP BY source, region
	`, windowStart.Add(4*time.Minute), windowStart))
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, `
		INSERT INTO default.anomaly_source_watermarks_v1 (source, region, time) VALUES
			('requests', 'anomaly-integration', ?),
			('resources', 'anomaly-integration', ?),
			('instance_events', 'anomaly-integration', ?)
	`, windowStart, windowStart.Add(4*time.Minute), windowStart))
	key := "deploy-anomaly-" + strconv.FormatInt(windowStart.Unix(), 10)
	_, err := hydrav1.NewCronServiceIngressClient(h.Restate, key).
		RunDeployAnomalyCheck().Request(h.Ctx, &hydrav1.RunDeployAnomalyCheckRequest{})
	require.NoError(t, err)
}

func uniqueAnomalyWindowStart() time.Time {
	const bucketsPerYear = 365 * 24 * 12
	offset := city.CH64([]byte(uid.New("window")))%bucketsPerYear + 1
	return time.Now().UTC().Truncate(5 * time.Minute).Add(time.Duration(offset) * 5 * time.Minute)
}
