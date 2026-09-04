package cron_test

import (
	"database/sql"
	"encoding/json"
	"math"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/go-faster/city"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

type anomalyTestApp struct {
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	deploymentID  string
	regionID      string
	appCreatedAt  int64
}

func TestRunDeployAnomalyCheck_Integration(t *testing.T) {
	h := harness.New(t)
	production := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
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
			resolution_message, observed_value, baseline_mean,
			baseline_stddev, threshold_sigma, window_start, window_end,
			created_at, updated_at
		FROM alert_events WHERE workspace_id = ?`, production.workspaceID).
		Scan(
			&alert.Pk, &alert.ID, &alert.WorkspaceID, &alert.ProjectID, &alert.AppID,
			&alert.EnvironmentID, &alert.DeploymentID, &alert.Metric, &alert.Status,
			&alert.FiredAt, &alert.LastSeenAt, &alert.ResolvedAt, &alert.ResolutionMessage,
			&alert.ObservedValue, &alert.BaselineMean,
			&alert.BaselineStddev, &alert.ThresholdSigma, &alert.WindowStart,
			&alert.WindowEnd, &alert.CreatedAt, &alert.UpdatedAt,
		)
	require.NoError(t, err)
	require.Equal(t, db.AlertEventsMetricRequests, alert.Metric)
	require.Equal(t, db.AlertEventsStatusOpen, alert.Status)
	require.Equal(t, 10_000.0, alert.ObservedValue)
	expectedMean := 12_000.0 / 288
	expectedStddev := math.Sqrt(12*1_000_000.0/288 - expectedMean*expectedMean)
	require.InDelta(t, expectedMean, alert.BaselineMean, 1e-9)
	require.InDelta(t, expectedStddev, alert.BaselineStddev, 1e-9)
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

	quietStart := assertIncompleteTelemetryNoop(t, h, production, alert.ID, windowStart)
	for i := 1; i <= 3; i++ {
		quietWindow := quietStart.Add(time.Duration(i) * 5 * time.Minute)
		insertAnomalyRequestBucket(t, h, production, quietWindow, 500)
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

	var resolutionMessage sql.NullString
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT resolution_message FROM alert_events WHERE id = ?", alert.ID).
		Scan(&resolutionMessage))
	require.Equal(t, "Metric returned to baseline for 3 consecutive windows", resolutionMessage.String)
	assertStoppedDeploymentSuppression(t, h)
	assertBaselineAdaptedResolution(t, h)
	assertDeploymentTopologyMetadata(t, h)
	assertAnomalyShardCompatibility(t, h)
	assertInstanceEventRecoveryWithoutNewEvents(t, h)
	assertOutOfOrderAnomalyWindowsIgnored(t, h)
	assertOrphanAlertSelfHeals(t, h, windowStart.Add(2*time.Hour))
}

func assertOrphanAlertSelfHeals(t *testing.T, h *harness.Harness, windowStart time.Time) {
	t.Helper()
	alertID := uid.New(uid.AlertPrefix)
	require.NoError(t, h.DB.InsertAlertEvent(h.Ctx, db.InsertAlertEventParams{
		ID: alertID, WorkspaceID: uid.New(uid.WorkspacePrefix), ProjectID: uid.New(uid.ProjectPrefix),
		AppID: uid.New("app"), EnvironmentID: uid.New("env"), Metric: db.AlertEventsMetricRequests,
		FiredAt: windowStart.UnixMilli(), LastSeenAt: windowStart.UnixMilli(),
		ObservedValue: 10_000, BaselineMean: 1_000, BaselineStddev: 20, ThresholdSigma: 4,
		WindowStart: windowStart.UnixMilli(), WindowEnd: windowStart.Add(5 * time.Minute).UnixMilli(),
		CreatedAt: windowStart.UnixMilli(), UpdatedAt: sql.NullInt64{},
	}))

	runAnomalyWindow(t, h, windowStart)

	var status string
	var resolvedAt sql.NullInt64
	var resolutionMessage sql.NullString
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx, `
		SELECT status, resolved_at, resolution_message
		FROM alert_events
		WHERE id = ?
	`, alertID).Scan(&status, &resolvedAt, &resolutionMessage))
	require.Equal(t, "resolved", status)
	require.True(t, resolvedAt.Valid)
	require.Equal(t, "Deployment stopped", resolutionMessage.String)
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
		anomalyIngressKey(openApp))
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
		anomalyIngressKey(newApp))
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
			DeploymentHasRunningRegion: true,
			Metrics:                    []*hydrav1.DeployAnomalyMetricInput{incomplete},
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
		anomalyIngressKey(app))
	request := &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowStart.Add(5 * time.Minute).UnixMilli(),
		WindowEnd:   windowStart.Add(10 * time.Minute).UnixMilli(),
		WorkspaceId: app.workspaceID, ProjectId: app.projectID,
		AppId: app.appID, EnvironmentId: app.environmentID,
		DeploymentId: app.deploymentID, DeploymentDesiredState: "stopped",
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

func assertBaselineAdaptedResolution(t *testing.T, h *harness.Harness) {
	t.Helper()
	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	windowEnd := time.Now().UTC().Truncate(5 * time.Minute)
	firedAt := windowEnd.Add(-24 * time.Hour)
	alertID := uid.New(uid.AlertPrefix)
	require.NoError(t, h.DB.InsertAlertEvent(h.Ctx, db.InsertAlertEventParams{
		ID: alertID, WorkspaceID: app.workspaceID, ProjectID: app.projectID,
		AppID: app.appID, EnvironmentID: app.environmentID,
		DeploymentID: sql.NullString{String: app.deploymentID, Valid: true},
		Metric:       db.AlertEventsMetricRequests, FiredAt: firedAt.UnixMilli(),
		LastSeenAt: firedAt.UnixMilli(), ObservedValue: 200, BaselineMean: 100,
		BaselineStddev: 10, ThresholdSigma: 4,
		WindowStart: firedAt.Add(-5 * time.Minute).UnixMilli(), WindowEnd: firedAt.UnixMilli(),
		CreatedAt: firedAt.UnixMilli(), UpdatedAt: sql.NullInt64{},
	}))

	response, err := hydrav1.NewDeployAnomalyServiceIngressClient(h.Restate,
		anomalyIngressKey(app)).
		Evaluate().Request(h.Ctx, &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowEnd.Add(-5 * time.Minute).UnixMilli(), WindowEnd: windowEnd.UnixMilli(),
		WorkspaceId: app.workspaceID, ProjectId: app.projectID,
		AppId: app.appID, EnvironmentId: app.environmentID,
		DeploymentId: app.deploymentID, DeploymentDesiredState: "running",
		Metrics: []*hydrav1.DeployAnomalyMetricInput{{
			Metric:    string(db.AlertEventsMetricRequests),
			DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
			Current:   200, BaselineMean: 200, ObservedBaselineBuckets: 288,
		}},
	})
	require.NoError(t, err)
	require.False(t, response.GetPending(), "max-age resolution must clear metric state")

	var status string
	var resolutionMessage sql.NullString
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT status, resolution_message FROM alert_events WHERE id = ?", alertID).
		Scan(&status, &resolutionMessage))
	require.Equal(t, "resolved", status)
	require.Equal(t, "Baseline adapted after 24 hours", resolutionMessage.String)
}

func assertDeploymentTopologyMetadata(t *testing.T, h *harness.Harness) {
	t.Helper()
	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	groupJSON, err := json.Marshal([]map[string]string{{
		"workspace_id": app.workspaceID, "project_id": app.projectID,
		"app_id": app.appID, "environment_id": app.environmentID,
	}})
	require.NoError(t, err)
	find := func() db.FindLiveDeploymentsForEnvironmentsRow {
		rows, findErr := h.DB.FindLiveDeploymentsForEnvironments(h.Ctx, db.FindLiveDeploymentsForEnvironmentsParams{
			GroupKeysJson: string(groupJSON),
		})
		require.NoError(t, findErr)
		require.Len(t, rows, 1)
		return rows[0]
	}

	require.Equal(t, app.appCreatedAt, find().AppCreatedAt)
	require.True(t, find().DeploymentHasRunningRegion)
	secondRegion := uid.New(uid.RegionPrefix)
	require.NoError(t, h.DB.InsertDeploymentTopology(h.Ctx, db.InsertDeploymentTopologyParams{
		WorkspaceID: app.workspaceID, DeploymentID: app.deploymentID, RegionID: secondRegion,
		AutoscalingReplicasMin: 1, AutoscalingReplicasMax: 1,
		DesiredStatus: db.DeploymentTopologyDesiredStatusStopped, CreatedAt: time.Now().UnixMilli(),
	}))
	require.True(t, find().DeploymentHasRunningRegion, "one running region keeps drop detection active")
	require.NoError(t, h.DB.UpdateDeploymentTopologyDesiredStatus(h.Ctx, db.UpdateDeploymentTopologyDesiredStatusParams{
		DesiredStatus: db.DeploymentTopologyDesiredStatusStopped,
		UpdatedAt:     sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		DeploymentID:  app.deploymentID, RegionID: app.regionID,
	}))
	require.False(t, find().DeploymentHasRunningRegion, "all stopped regions suppress request-drop alerts")
}

func assertAnomalyShardCompatibility(t *testing.T, h *harness.Harness) {
	t.Helper()
	vectors := map[string]uint64{
		"ws_test":  14_195_424_828_609_858_884,
		"ws_local": 6_295_136_450_341_388_082,
	}
	appID := uid.New("app")
	for workspaceID, expectedHash := range vectors {
		hash := city.CH64([]byte(workspaceID))
		require.Equal(t, expectedHash, hash)
		var clickhouseHash uint64
		require.NoError(t, h.ClickHouseConn.QueryRow(h.Ctx, "SELECT cityHash64(?)", workspaceID).Scan(&clickhouseHash))
		require.Equal(t, hash, clickhouseHash)
		require.NoError(t, h.DB.InsertAlertEvent(h.Ctx, db.InsertAlertEventParams{
			ID: uid.New(uid.AlertPrefix), WorkspaceID: workspaceID,
			ProjectID: "project", AppID: appID, EnvironmentID: "environment",
			Metric: db.AlertEventsMetricRequests, FiredAt: 1, LastSeenAt: 1,
			ObservedValue: 1, BaselineMean: 0, BaselineStddev: 0, ThresholdSigma: 4,
			WindowStart: 0, WindowEnd: 1, CreatedAt: 1,
		}))
	}

	rows, err := h.DB.ListOpenAlertEventGroups(h.Ctx)
	require.NoError(t, err)
	seen := make(map[string]int)
	for shard := range uint64(16) {
		for _, row := range rows {
			expectedHash, ok := vectors[row.WorkspaceID]
			if !ok || row.AppID != appID {
				continue
			}
			if city.CH64([]byte(row.WorkspaceID))%16 != shard {
				continue
			}
			require.Equal(t, shard, expectedHash%16)
			seen[row.WorkspaceID]++
		}
	}
	for workspaceID := range vectors {
		require.Equal(t, 1, seen[workspaceID])
	}
}

func assertInstanceEventRecoveryWithoutNewEvents(t *testing.T, h *harness.Harness) {
	t.Helper()

	recoveryApp := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	recoveryStart := uniqueAnomalyWindowStart()
	insertAnomalyOOMEvent(t, h, recoveryApp, recoveryStart)
	advanceResourceWatermark(t, h, recoveryApp, recoveryStart)
	runAnomalyWindow(t, h, recoveryStart)
	recoveryAlertID := findOpenAnomalyAlertID(t, h, recoveryApp, db.AlertEventsMetricOomKilled)
	for i := 1; i <= 3; i++ {
		window := recoveryStart.Add(time.Duration(i) * 5 * time.Minute)
		advanceResourceWatermark(t, h, recoveryApp, window)
		runAnomalyWindow(t, h, window)
	}
	requireAnomalyAlertResolution(t, h, recoveryAlertID, "Metric returned to baseline for 3 consecutive windows")

	maxAgeApp := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	maxAgeStart := time.Now().UTC().Truncate(5 * time.Minute).Add(-25 * time.Hour)
	insertAnomalyOOMEvent(t, h, maxAgeApp, maxAgeStart)
	advanceResourceWatermark(t, h, maxAgeApp, maxAgeStart)
	runAnomalyWindow(t, h, maxAgeStart)
	maxAgeAlertID := findOpenAnomalyAlertID(t, h, maxAgeApp, db.AlertEventsMetricOomKilled)
	maxAgeWindow := maxAgeStart.Add(24 * time.Hour)
	advanceResourceWatermark(t, h, maxAgeApp, maxAgeWindow)
	runAnomalyWindow(t, h, maxAgeWindow)
	requireAnomalyAlertResolution(t, h, maxAgeAlertID, "Baseline adapted after 24 hours")
}

func anomalyIngressKey(app anomalyTestApp) string {
	return url.PathEscape(deployanomaly.GroupKey(app.workspaceID, app.appID, app.environmentID))
}

func assertOutOfOrderAnomalyWindowsIgnored(t *testing.T, h *harness.Harness) {
	t.Helper()

	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	windowOne := uniqueAnomalyWindowStart()
	anomalous := &hydrav1.DeployAnomalyMetricInput{
		Metric:    string(db.AlertEventsMetricError5xx),
		DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
		Current:   30, RequestsInWindow: 100, BaselineMean: 0.01,
		ObservedBaselineBuckets: 12,
	}
	recovered := &hydrav1.DeployAnomalyMetricInput{
		Metric:    string(db.AlertEventsMetricError5xx),
		DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
		Current:   1, RequestsInWindow: 100, BaselineMean: 0.01,
		ObservedBaselineBuckets: 12,
	}

	sendAnomalyMetric(t, h, app, windowOne.Add(5*time.Minute), anomalous)
	sendAnomalyMetric(t, h, app, windowOne, recovered)
	sendAnomalyMetric(t, h, app, windowOne.Add(10*time.Minute), anomalous)
	alertID := findOpenAnomalyAlertID(t, h, app, db.AlertEventsMetricError5xx)

	touchWindow := windowOne.Add(15 * time.Minute)
	sendAnomalyMetric(t, h, app, touchWindow, anomalous)
	sendAnomalyMetric(t, h, app, windowOne.Add(5*time.Minute), anomalous)
	var lastSeenAt int64
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT last_seen_at FROM alert_events WHERE id = ?", alertID).Scan(&lastSeenAt))
	require.Equal(t, touchWindow.Add(5*time.Minute).UnixMilli(), lastSeenAt)

	sendAnomalyMetric(t, h, app, windowOne.Add(20*time.Minute), recovered)
	sendAnomalyMetric(t, h, app, touchWindow, recovered)
	sendAnomalyMetric(t, h, app, windowOne.Add(25*time.Minute), recovered)
	var status string
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT status FROM alert_events WHERE id = ?", alertID).Scan(&status))
	require.Equal(t, "open", status, "a stale recovery must not advance the quiet-window count")
	sendAnomalyMetric(t, h, app, windowOne.Add(30*time.Minute), recovered)
	requireAnomalyAlertResolution(t, h, alertID, "Metric returned to baseline for 3 consecutive windows")
}

func sendAnomalyMetric(
	t *testing.T,
	h *harness.Harness,
	app anomalyTestApp,
	windowStart time.Time,
	metric *hydrav1.DeployAnomalyMetricInput,
) {
	t.Helper()
	_, err := hydrav1.NewDeployAnomalyServiceIngressClient(h.Restate,
		anomalyIngressKey(app)).
		Evaluate().Request(h.Ctx, &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowStart.UnixMilli(), WindowEnd: windowStart.Add(5 * time.Minute).UnixMilli(),
		WorkspaceId: app.workspaceID, ProjectId: app.projectID,
		AppId: app.appID, EnvironmentId: app.environmentID,
		DeploymentId: app.deploymentID, DeploymentDesiredState: "running",
		DeploymentHasRunningRegion: true,
		AppCreatedAt:               app.appCreatedAt,
		Metrics:                    []*hydrav1.DeployAnomalyMetricInput{metric},
	})
	require.NoError(t, err)
}

func insertAnomalyOOMEvent(t *testing.T, h *harness.Harness, app anomalyTestApp, windowStart time.Time) {
	t.Helper()
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, `
		INSERT INTO default.instance_events_raw_v1 (
			time, workspace_id, project_id, app_id, environment_id, deployment_id,
			event_kind, reason, region, event_fingerprint, attributes
		) VALUES (?, ?, ?, ?, ?, ?, 'terminated', 'OOMKilled', 'anomaly-integration', ?, '{}')
	`, windowStart.Add(time.Minute).UnixMilli(), app.workspaceID, app.projectID,
		app.appID, app.environmentID, app.deploymentID, uid.New("event")))
}

func findOpenAnomalyAlertID(
	t *testing.T,
	h *harness.Harness,
	app anomalyTestApp,
	metric db.AlertEventsMetric,
) string {
	t.Helper()
	var alertID string
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx, `
		SELECT id FROM alert_events
		WHERE app_id = ? AND environment_id = ? AND metric = ? AND status = 'open'
	`, app.appID, app.environmentID, metric).Scan(&alertID))
	return alertID
}

func requireAnomalyAlertResolution(t *testing.T, h *harness.Harness, alertID, message string) {
	t.Helper()
	var status string
	var resolutionMessage sql.NullString
	require.NoError(t, h.DB.RO().QueryRowContext(h.Ctx,
		"SELECT status, resolution_message FROM alert_events WHERE id = ?", alertID).
		Scan(&status, &resolutionMessage))
	require.Equal(t, "resolved", status)
	require.Equal(t, message, resolutionMessage.String)
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
	appCreatedAt := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
	_, err := h.DB.RW().ExecContext(h.Ctx, "UPDATE apps SET created_at = ? WHERE id = ?", appCreatedAt, app.ID)
	require.NoError(t, err)
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
	regionID := uid.New(uid.RegionPrefix)
	require.NoError(t, h.DB.InsertDeploymentTopology(h.Ctx, db.InsertDeploymentTopologyParams{
		WorkspaceID: workspace.ID, DeploymentID: deployment.ID, RegionID: regionID,
		AutoscalingReplicasMin: 0, AutoscalingReplicasMax: 1,
		DesiredStatus: db.DeploymentTopologyDesiredStatusRunning, CreatedAt: time.Now().UnixMilli(),
	}))
	return anomalyTestApp{
		workspaceID: workspace.ID, projectID: project.ID, appID: app.ID,
		environmentID: environment.ID, deploymentID: deployment.ID, regionID: regionID,
		appCreatedAt: appCreatedAt,
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
	watermark := time.Now().UTC().Truncate(5 * time.Minute)
	require.NoError(t, h.ClickHouseConn.Exec(h.Ctx, `
		INSERT INTO default.anomaly_source_watermarks_v1 (source, region, time) VALUES
			('requests', 'anomaly-integration', ?),
			('resources', 'anomaly-integration', ?)
	`, watermark, watermark.Add(4*time.Minute)))
	key := "deploy-anomaly-" + strconv.FormatInt(windowStart.Unix(), 10)
	_, err := hydrav1.NewCronServiceIngressClient(h.Restate, key).
		RunDeployAnomalyCheck().Request(h.Ctx, &hydrav1.RunDeployAnomalyCheckRequest{})
	require.NoError(t, err)
}

func uniqueAnomalyWindowStart() time.Time {
	offset := city.CH64([]byte(uid.New("window"))) % 96
	return time.Now().UTC().Truncate(5 * time.Minute).Add(-12*time.Hour + time.Duration(offset)*5*time.Minute)
}
