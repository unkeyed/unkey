package deployanomaly_test

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deployanomaly"
)

type recoveryFailureDB struct {
	db.Database

	failTouch     atomic.Bool
	touchAttempts atomic.Int32
	mu            sync.Mutex
	resolved      map[string]bool
}

func (d *recoveryFailureDB) FindOpenAlertEventsByGroup(
	context.Context,
	db.FindOpenAlertEventsByGroupParams,
) ([]db.AlertEvent, error) {
	firedAt := time.Now().Add(-2 * time.Hour).UnixMilli()
	return []db.AlertEvent{
		{
			ID: "alert_5xx", Metric: db.AlertEventsMetricError5xx, FiredAt: firedAt,
			ObservedValue: 0.5, BaselineMean: 0.01, BaselineStddev: 0, ThresholdSigma: 4,
		},
		{
			ID: "alert_4xx", Metric: db.AlertEventsMetricError4xx, FiredAt: firedAt,
			ObservedValue: 0.5, BaselineMean: 0.01, BaselineStddev: 0, ThresholdSigma: 4,
		},
	}, nil
}

func (d *recoveryFailureDB) TouchAlertEventLastSeen(
	_ context.Context,
	_ db.TouchAlertEventLastSeenParams,
) error {
	d.touchAttempts.Add(1)
	if d.failTouch.Load() {
		return errors.New("injected touch failure")
	}
	return nil
}

func (d *recoveryFailureDB) ResolveAlertEventBySystem(
	_ context.Context,
	params db.ResolveAlertEventBySystemParams,
) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.resolved[params.ID] = true
	return 1, nil
}

func (d *recoveryFailureDB) isResolved(alertID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.resolved[alertID]
}

func TestKilledWindowDoesNotDoubleCountQuietMetricOnRedispatch(t *testing.T) {
	database := &recoveryFailureDB{resolved: make(map[string]bool)}
	database.failTouch.Store(true)
	handler, err := deployanomaly.NewCheckHandler(deployanomaly.CheckConfig{DB: database})
	require.NoError(t, err)
	retry := restate.WithInvocationRetryPolicy(
		restate.WithInitialInterval(10*time.Millisecond),
		restate.WithMaxInterval(10*time.Millisecond),
		restate.WithMaxAttempts(2),
		restate.KillOnMaxAttempts(),
	)
	testEnv := containers.Restate(t,
		hydrav1.NewDeployAnomalyServiceServer(handler).ConfigureHandler("Evaluate", retry),
	)
	client := hydrav1.NewDeployAnomalyServiceIngressClient(testEnv.IngressClient, url.PathEscape("ws/app/env"))
	windowStart := time.Now().UTC().Add(-time.Hour).Truncate(5 * time.Minute)
	request := recoveryRequest(windowStart)

	_, err = client.Evaluate().Request(t.Context(), request)
	require.Error(t, err)
	require.Equal(t, int32(2), database.touchAttempts.Load())
	require.False(t, database.isResolved("alert_5xx"))

	database.failTouch.Store(false)
	_, err = client.Evaluate().Request(t.Context(), request)
	require.NoError(t, err)

	request = recoveryRequest(windowStart.Add(5 * time.Minute))
	_, err = client.Evaluate().Request(t.Context(), request)
	require.NoError(t, err)
	require.False(t, database.isResolved("alert_5xx"), "two distinct quiet windows must not resolve")

	request = recoveryRequest(windowStart.Add(10 * time.Minute))
	_, err = client.Evaluate().Request(t.Context(), request)
	require.NoError(t, err)
	require.True(t, database.isResolved("alert_5xx"), "three distinct quiet windows must resolve")
}

func recoveryRequest(windowStart time.Time) *hydrav1.EvaluateDeployAnomalyRequest {
	return &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowStart.UnixMilli(), WindowEnd: windowStart.Add(5 * time.Minute).UnixMilli(),
		WorkspaceId: "ws", ProjectId: "project", AppId: "app", EnvironmentId: "env",
		DeploymentId: "deployment", DeploymentDesiredState: "running", DeploymentHasRunningRegion: true,
		Metrics: []*hydrav1.DeployAnomalyMetricInput{
			{
				Metric:    string(db.AlertEventsMetricError5xx),
				DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
				Current:   1, RequestsInWindow: 100,
			},
			{
				Metric:    string(db.AlertEventsMetricError4xx),
				DataState: hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT,
				Current:   50, RequestsInWindow: 100,
			},
		},
	}
}
