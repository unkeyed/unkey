package cron_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/harness"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

type eventTestCron struct {
	hydrav1.UnimplementedCronServiceServer
	handler *deployanomaly.EventsHandler
}

func (s *eventTestCron) RunDeployAnomalyEvents(ctx restate.ObjectContext, req *hydrav1.RunDeployAnomalyEventsRequest) (*hydrav1.RunDeployAnomalyEventsResponse, error) {
	return s.handler.Handle(ctx, req)
}

func startEventWorker(t *testing.T, database db.Database, workspaces []string) containers.RestateConfig {
	t.Helper()
	check, err := deployanomaly.NewCheckHandler(deployanomaly.CheckConfig{DB: database, FastWorkspaces: workspaces})
	require.NoError(t, err)
	poll, err := deployanomaly.NewEventsHandler(deployanomaly.EventsConfig{
		DB: database, Workspaces: workspaces, Heartbeat: healthcheck.NewNoop(),
	})
	require.NoError(t, err)
	return containers.Restate(t,
		hydrav1.NewDeployAnomalyServiceServer(check).
			ConfigureHandler("Evaluate", deployanomaly.EventsRetryPolicy()).
			ConfigureHandler("OpenObservedEvents", deployanomaly.EventsRetryPolicy()),
		hydrav1.NewCronServiceServer(&eventTestCron{handler: poll}).
			ConfigureHandler("RunDeployAnomalyEvents", deployanomaly.EventsRetryPolicy()),
	)
}

func insertInboxEvent(t *testing.T, h *harness.Harness, app anomalyTestApp, metric db.DeployAnomalyEventsMetric) string {
	t.Helper()
	id := uid.New("event")
	t.Cleanup(func() {
		_, err := h.DB.RW().ExecContext(context.Background(), "DELETE FROM deploy_anomaly_events WHERE id = ?", id)
		require.NoError(t, err)
	})
	require.NoError(t, h.DB.InsertDeployAnomalyEvent(h.Ctx, db.InsertDeployAnomalyEventParams{
		ID: id, WorkspaceID: app.workspaceID, ProjectID: app.projectID, AppID: app.appID,
		EnvironmentID: app.environmentID, DeploymentID: app.deploymentID, Metric: metric,
		EventTime: time.Now().Add(-4 * time.Hour).UnixMilli(), ReceivedAt: time.Now().UnixMilli(),
	}))
	return id
}

func inboxRequest(app anomalyTestApp, ids ...string) *hydrav1.OpenObservedDeployAnomalyEventsRequest {
	return &hydrav1.OpenObservedDeployAnomalyEventsRequest{
		Group: &hydrav1.DeployAnomalyGroupKey{
			WorkspaceId: app.workspaceID, ProjectId: app.projectID, AppId: app.appID, EnvironmentId: app.environmentID,
		}, EventIds: ids,
	}
}

func eventQuietWindow(app anomalyTestApp, start time.Time) *hydrav1.EvaluateDeployAnomalyRequest {
	return &hydrav1.EvaluateDeployAnomalyRequest{
		WorkspaceId: app.workspaceID, ProjectId: app.projectID, AppId: app.appID, EnvironmentId: app.environmentID,
		DeploymentId: app.deploymentID, DeploymentDesiredState: "running", DeploymentHasRunningRegion: true,
		WindowStart: start.UnixMilli(), WindowEnd: start.Add(5 * time.Minute).UnixMilli(),
		Metrics: []*hydrav1.DeployAnomalyMetricInput{
			{Metric: "oom_killed", DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE},
			{Metric: "crash_loop", DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE},
		},
	}
}

func countEventAlerts(t *testing.T, h *harness.Harness, app anomalyTestApp, status string) int {
	t.Helper()
	var count int
	require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM alert_events WHERE app_id = ? AND status = ?", app.appID, status).Scan(&count))
	return count
}

func TestDeployAnomalyEventsLifecycle_Integration(t *testing.T) {
	h := harness.New(t)
	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	preview := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindPreview)
	off := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	worker := startEventWorker(t, h.DB, []string{app.workspaceID, preview.workspaceID})
	oom := insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
	crash := insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricCrashLoop)
	insertInboxEvent(t, h, preview, db.DeployAnomalyEventsMetricOomKilled)
	insertInboxEvent(t, h, off, db.DeployAnomalyEventsMetricOomKilled)
	start := time.Now()
	result, err := hydrav1.NewCronServiceIngressClient(worker.IngressClient, "deploy-anomaly-events").
		RunDeployAnomalyEvents().Request(h.Ctx, &hydrav1.RunDeployAnomalyEventsRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(3), result.GetEventsProcessed())
	require.Equal(t, int32(2), result.GetAlertsOpened())
	require.Equal(t, 2, countEventAlerts(t, h, app, "open"))
	require.Zero(t, countEventAlerts(t, h, preview, "open"))
	require.Zero(t, countEventAlerts(t, h, off, "open"))
	t.Logf("late event accepted: poll-to-persist=%s, event_age=4h, opened=2", time.Since(start))

	client := hydrav1.NewDeployAnomalyServiceIngressClient(worker.IngressClient, anomalyIngressKey(app))
	quietStart := time.Now().UTC().Truncate(5 * time.Minute).Add(5 * time.Minute)
	_, err = client.Evaluate().Request(h.Ctx, eventQuietWindow(app, quietStart.Add(-10*time.Minute)))
	require.NoError(t, err)
	_, err = client.Evaluate().Request(h.Ctx, eventQuietWindow(app, quietStart))
	require.NoError(t, err)
	duplicate, err := client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, oom, crash))
	require.NoError(t, err)
	require.Zero(t, duplicate.GetEventsProcessed())
	incomplete := eventQuietWindow(app, quietStart.Add(5*time.Minute))
	for _, metric := range incomplete.Metrics {
		metric.DataState = hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE
	}
	_, err = client.Evaluate().Request(h.Ctx, incomplete)
	require.NoError(t, err)
	require.Equal(t, 2, countEventAlerts(t, h, app, "open"))
	for i := 1; i <= 2; i++ {
		_, err = client.Evaluate().Request(h.Ctx, eventQuietWindow(app, quietStart.Add(time.Duration(i)*5*time.Minute)))
		require.NoError(t, err)
		if i == 1 {
			require.Equal(t, 2, countEventAlerts(t, h, app, "open"))
		}
	}
	require.Equal(t, 2, countEventAlerts(t, h, app, "resolved"))
	duplicate, err = client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, oom, crash))
	require.NoError(t, err)
	require.Zero(t, duplicate.GetAlertsOpened())
	require.Zero(t, countEventAlerts(t, h, app, "open"))
	newLife := insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
	opened, err := client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, newLife))
	require.NoError(t, err)
	require.Equal(t, int32(1), opened.GetAlertsOpened())
	require.Equal(t, 1, countEventAlerts(t, h, app, "open"))
}

type eventReconcileFailureDB struct {
	db.Database
	fail atomic.Bool
}

func (d *eventReconcileFailureDB) FindOpenAlertEventsByGroup(ctx context.Context, params db.FindOpenAlertEventsByGroupParams) ([]db.AlertEvent, error) {
	if d.fail.Load() {
		return nil, errors.New("injected post-commit reconciliation failure")
	}
	return d.Database.FindOpenAlertEventsByGroup(ctx, params)
}

func TestDeployAnomalyEventsKill_Integration(t *testing.T) {
	h := harness.New(t)
	for _, phase := range []string{"pre-insert", "mixed-batch", "post-commit"} {
		t.Run(phase, func(t *testing.T) {
			app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
			database := &eventReconcileFailureDB{Database: h.DB}
			check, err := deployanomaly.NewCheckHandler(deployanomaly.CheckConfig{DB: database, FastWorkspaces: []string{app.workspaceID}})
			require.NoError(t, err)
			retry := restate.WithInvocationRetryPolicy(
				restate.WithInitialInterval(10*time.Millisecond), restate.WithMaxInterval(10*time.Millisecond),
				restate.WithMaxAttempts(2), restate.KillOnMaxAttempts(),
			)
			worker := containers.Restate(t, hydrav1.NewDeployAnomalyServiceServer(check).
				ConfigureHandler("OpenObservedEvents", retry).ConfigureHandler("Evaluate", retry))
			ids := []string{insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)}
			triggerName := uid.New("fail_anomaly")
			if phase == "post-commit" {
				database.fail.Store(true)
			} else {
				metric := "oom_killed"
				if phase == "mixed-batch" {
					metric = "crash_loop"
					ids = append(ids, insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricCrashLoop))
				}
				_, err = h.DB.RW().ExecContext(h.Ctx, fmt.Sprintf(`CREATE TRIGGER %s BEFORE INSERT ON alert_events FOR EACH ROW
BEGIN IF NEW.workspace_id = '%s' AND NEW.metric = '%s' THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected insert failure'; END IF; END`, triggerName, app.workspaceID, metric))
				require.NoError(t, err)
				t.Cleanup(func() {
					_, err := h.DB.RW().ExecContext(context.Background(), "DROP TRIGGER IF EXISTS "+triggerName)
					require.NoError(t, err)
				})
			}
			client := hydrav1.NewDeployAnomalyServiceIngressClient(worker.IngressClient, anomalyIngressKey(app))
			_, err = client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, ids...))
			require.Error(t, err)
			var processed int
			require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx,
				"SELECT COUNT(*) FROM deploy_anomaly_events WHERE workspace_id = ? AND processed_at IS NOT NULL", app.workspaceID).Scan(&processed))
			if phase == "post-commit" {
				require.Equal(t, 1, processed)
				require.Equal(t, 1, countEventAlerts(t, h, app, "open"))
				database.fail.Store(false)
				quiet := time.Now().UTC().Truncate(5 * time.Minute).Add(5 * time.Minute)
				for i := range 3 {
					_, err = client.Evaluate().Request(h.Ctx, eventQuietWindow(app, quiet.Add(time.Duration(i)*5*time.Minute)))
					require.NoError(t, err)
				}
				require.Equal(t, 1, countEventAlerts(t, h, app, "resolved"))
			} else {
				require.Zero(t, processed)
				require.Zero(t, countEventAlerts(t, h, app, "open"))
				_, err = h.DB.RW().ExecContext(h.Ctx, "DROP TRIGGER "+triggerName)
				require.NoError(t, err)
			}
			result, err := client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, ids...))
			require.NoError(t, err)
			if phase == "post-commit" {
				require.Zero(t, result.GetEventsProcessed())
				require.Zero(t, countEventAlerts(t, h, app, "open"), "replay after recovery must not reopen")
			} else {
				require.Equal(t, int32(len(ids)), result.GetEventsProcessed())
				require.Equal(t, len(ids), countEventAlerts(t, h, app, "open"))
			}
			duplicate, err := client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, ids...))
			require.NoError(t, err)
			require.Zero(t, duplicate.GetEventsProcessed())
			t.Logf("%s: killed invocation, atomic SQL outcome, fresh dispatch and duplicate verified", phase)
		})
	}
}

func TestDeployAnomalyEventsBoundedSweep_Integration(t *testing.T) {
	h := harness.New(t)
	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	worker := startEventWorker(t, h.DB, []string{app.workspaceID})
	for range 1_001 {
		insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
	}
	client := hydrav1.NewCronServiceIngressClient(worker.IngressClient, "deploy-anomaly-events")
	start := time.Now()
	first, err := client.RunDeployAnomalyEvents().Request(h.Ctx, &hydrav1.RunDeployAnomalyEventsRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1_000), first.GetEventsProcessed())
	require.Equal(t, int32(1), first.GetAlertsOpened())
	insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
	second, err := client.RunDeployAnomalyEvents().Request(h.Ctx, &hydrav1.RunDeployAnomalyEventsRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), second.GetEventsProcessed(), "a sweep must finish at its original high-water mark")
	third, err := client.RunDeployAnomalyEvents().Request(h.Ctx, &hydrav1.RunDeployAnomalyEventsRequest{})
	require.NoError(t, err)
	require.Equal(t, int32(1), third.GetEventsProcessed(), "new arrivals belong to the next sweep")
	require.Equal(t, 1, countEventAlerts(t, h, app, "open"))
	var pending int
	require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx,
		"SELECT COUNT(*) FROM deploy_anomaly_events WHERE workspace_id = ? AND processed_at IS NULL", app.workspaceID).Scan(&pending))
	require.Zero(t, pending)
	t.Logf("1002 events, three bounded polls, one alert: %s", time.Since(start))
}

func TestDeployAnomalyEventsSuppression_Integration(t *testing.T) {
	h := harness.New(t)
	for _, state := range []string{"disabled", "stopped", "no-running-region", "rollover", "missing-deployment"} {
		t.Run(state, func(t *testing.T) {
			app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
			id := insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
			workspaces := []string{app.workspaceID}
			switch state {
			case "disabled":
				workspaces = nil
			case "stopped":
				_, err := h.DB.RW().ExecContext(h.Ctx, "UPDATE deployments SET desired_state = 'stopped' WHERE id = ?", app.deploymentID)
				require.NoError(t, err)
			case "no-running-region":
				_, err := h.DB.RW().ExecContext(h.Ctx, "UPDATE deployment_topology SET desired_status = 'stopped' WHERE deployment_id = ?", app.deploymentID)
				require.NoError(t, err)
			case "rollover":
				_, err := h.DB.RW().ExecContext(h.Ctx, "UPDATE deploy_anomaly_events SET deployment_id = 'previous-deployment' WHERE id = ?", id)
				require.NoError(t, err)
			case "missing-deployment":
				_, err := h.DB.RW().ExecContext(h.Ctx, "UPDATE apps SET current_deployment_id = NULL WHERE id = ?", app.appID)
				require.NoError(t, err)
			}
			worker := startEventWorker(t, h.DB, workspaces)
			result, err := hydrav1.NewDeployAnomalyServiceIngressClient(worker.IngressClient, anomalyIngressKey(app)).
				OpenObservedEvents().Request(h.Ctx, inboxRequest(app, id))
			require.NoError(t, err)
			require.Zero(t, result.GetAlertsOpened())
			require.Zero(t, countEventAlerts(t, h, app, "open"))
			if state == "disabled" {
				require.Zero(t, result.GetEventsProcessed())
			} else {
				require.Equal(t, int32(1), result.GetEventsProcessed())
			}
		})
	}
}

func TestDeployAnomalyEventsFreshFailureResetsRecovery_Integration(t *testing.T) {
	h := harness.New(t)
	app := createAnomalyTestApp(t, h, mysqltype.EnvironmentKindProduction)
	currentWindow := time.Now().UTC().Truncate(5 * time.Minute)
	firedAt := currentWindow.Add(-30 * time.Minute).UnixMilli()
	require.NoError(t, h.DB.InsertAlertEvent(h.Ctx, db.InsertAlertEventParams{
		ID: uid.New(uid.AlertPrefix), WorkspaceID: app.workspaceID, ProjectID: app.projectID,
		AppID: app.appID, EnvironmentID: app.environmentID, DeploymentID: sql.NullString{String: app.deploymentID, Valid: true},
		Metric: db.AlertEventsMetricOomKilled, FiredAt: firedAt, LastSeenAt: firedAt,
		LastObservedEventAt: sql.NullInt64{Int64: firedAt, Valid: true}, ObservedValue: 1,
		WindowStart: firedAt, WindowEnd: firedAt, CreatedAt: firedAt,
	}))
	worker := startEventWorker(t, h.DB, []string{app.workspaceID})
	client := hydrav1.NewDeployAnomalyServiceIngressClient(worker.IngressClient, anomalyIngressKey(app))
	for i := 2; i > 0; i-- {
		_, err := client.Evaluate().Request(h.Ctx, eventQuietWindow(app, currentWindow.Add(-time.Duration(i)*5*time.Minute)))
		require.NoError(t, err)
	}
	require.Equal(t, 1, countEventAlerts(t, h, app, "open"))
	id := insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
	result, err := client.OpenObservedEvents().Request(h.Ctx, inboxRequest(app, id))
	require.NoError(t, err)
	require.Equal(t, int32(1), result.GetEventsProcessed())
	require.Zero(t, result.GetAlertsOpened())
	incomplete := eventQuietWindow(app, currentWindow)
	for _, metric := range incomplete.Metrics {
		metric.DataState = hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE
	}
	_, err = client.Evaluate().Request(h.Ctx, incomplete)
	require.NoError(t, err)
	for i := 1; i <= 3; i++ {
		_, err = client.Evaluate().Request(h.Ctx, eventQuietWindow(app, currentWindow.Add(time.Duration(i)*5*time.Minute)))
		require.NoError(t, err)
		if i < 3 {
			require.Equal(t, 1, countEventAlerts(t, h, app, "open"), "quiet windows before the fresh failure cannot contribute to recovery")
		}
	}
	require.Equal(t, 1, countEventAlerts(t, h, app, "resolved"))
}

func TestDeployAnomalyEventsFleetWithProcessedHistory_Integration(t *testing.T) {
	h := harness.New(t)
	workspace := h.Seed.CreateWorkspace(h.Ctx)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		for _, table := range []string{"deploy_anomaly_events", "alert_events", "deployments", "environments", "apps", "projects"} {
			_, err := h.DB.RW().ExecContext(ctx, "DELETE FROM "+table+" WHERE workspace_id = ?", workspace.ID)
			require.NoError(t, err)
		}
	})
	apps := make([]anomalyTestApp, 100)
	for i := range apps {
		apps[i] = createAnomalyTestAppInWorkspace(t, h, mysqltype.EnvironmentKindProduction, workspace.ID)
	}
	_, err := h.DB.RW().ExecContext(h.Ctx, `INSERT INTO deploy_anomaly_events
    (id, workspace_id, project_id, app_id, environment_id, deployment_id, metric, event_time, received_at, processed_at)
WITH digits AS (SELECT 0 AS n UNION ALL SELECT 1 UNION ALL SELECT 2 UNION ALL SELECT 3 UNION ALL SELECT 4
    UNION ALL SELECT 5 UNION ALL SELECT 6 UNION ALL SELECT 7 UNION ALL SELECT 8 UNION ALL SELECT 9)
SELECT CONCAT(?, '-', a.n, b.n, c.n, d.n, e.n), ?, ?, ?, ?, ?, 'oom_killed', 1, 1, 2
FROM digits a CROSS JOIN digits b CROSS JOIN digits c CROSS JOIN digits d CROSS JOIN digits e`,
		uid.New("history"), workspace.ID, apps[0].projectID, apps[0].appID, apps[0].environmentID, apps[0].deploymentID)
	require.NoError(t, err)
	for _, app := range apps {
		insertInboxEvent(t, h, app, db.DeployAnomalyEventsMetricOomKilled)
	}
	maximum, err := h.DB.FindPendingDeployAnomalyEventMaxPk(h.Ctx, workspace.ID)
	require.NoError(t, err)
	var plan string
	require.NoError(t, h.DB.RW().QueryRowContext(h.Ctx, `EXPLAIN ANALYZE SELECT pk, id, workspace_id, project_id,
    app_id, environment_id, deployment_id, metric, event_time, received_at, processed_at
FROM deploy_anomaly_events WHERE workspace_id = ? AND processed_at IS NULL AND pk > 0 AND pk <= ? ORDER BY pk LIMIT 100`,
		workspace.ID, maximum).Scan(&plan))
	require.Contains(t, plan, "deploy_anomaly_events_pending_idx")
	t.Logf("pending page with 100000 processed identities: %s", plan)
	worker := startEventWorker(t, h.DB, []string{workspace.ID})
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	result, err := hydrav1.NewCronServiceIngressClient(worker.IngressClient, "deploy-anomaly-events").
		RunDeployAnomalyEvents().Request(h.Ctx, &hydrav1.RunDeployAnomalyEventsRequest{})
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	require.NoError(t, err)
	require.Equal(t, int32(100), result.GetEventsProcessed())
	require.Equal(t, int32(100), result.GetAlertsOpened())
	for _, app := range apps {
		require.Equal(t, 1, countEventAlerts(t, h, app, "open"))
	}
	t.Logf("100 affected apps, one workspace, 100000 processed identities: poll=%s, Go process allocations=%d bytes, heap_at_return=%d bytes",
		elapsed, after.TotalAlloc-before.TotalAlloc, after.HeapAlloc)
}
