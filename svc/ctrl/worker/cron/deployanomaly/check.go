package deployanomaly

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	autoResolveMessage     = "Metric returned to baseline for 3 consecutive windows"
	baselineAdaptedMessage = "Baseline adapted after 24 hours"
	stoppedMessage         = "Deployment stopped"
	lastWindowEndStateKey  = "last_window_end"
)

type CheckConfig struct {
	DB db.Database
}

type CheckHandler struct {
	db db.Database
}

func NewCheckHandler(cfg CheckConfig) (*CheckHandler, error) {
	if err := assert.NotNil(cfg.DB, "DB must not be nil"); err != nil {
		return nil, err
	}
	return &CheckHandler{db: cfg.DB}, nil
}

func candidateKey(metric Metric) string       { return "candidate:" + string(metric) }
func candidateWindowKey(metric Metric) string { return "candidate_window:" + string(metric) }
func openAlertKey(metric Metric) string       { return "open_alert:" + string(metric) }
func firedAtKey(metric Metric) string         { return "fired_at:" + string(metric) }
func quietKey(metric Metric) string           { return "quiet:" + string(metric) }
func snapshotKey(metric Metric) string        { return "snapshot:" + string(metric) }
func detectorInputKey(metric Metric) string   { return "detector_input:" + string(metric) }

var allMetrics = []Metric{
	MetricError5xx,
	MetricError4xx,
	MetricRequests,
	MetricRequestsDrop,
	MetricEgressBytes,
	MetricCPUSeconds,
	MetricMemoryUtilization,
	MetricOOMKilled,
	MetricCrashLoop,
}

// Evaluate applies complete metric windows to candidate and open-alert state.
// Incomplete sources are strict no-ops so ingest lag cannot generate or resolve
// a customer alert.
func (h *CheckHandler) Evaluate(
	ctx restate.ObjectContext,
	req *hydrav1.EvaluateDeployAnomalyRequest,
) (*hydrav1.EvaluateDeployAnomalyResponse, error) {
	lastWindowEnd, err := restate.Get[int64](ctx, lastWindowEndStateKey)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("get last deploy anomaly window"))
	}
	if req.GetWindowEnd() <= lastWindowEnd {
		logger.Warn("deploy anomaly window skipped because it is not newer",
			"workspace_id", req.GetWorkspaceId(), "app_id", req.GetAppId(),
			"environment_id", req.GetEnvironmentId(), "window_end", req.GetWindowEnd(),
			"last_window_end", lastWindowEnd,
		)
		pending, pendingErr := hasPendingState(ctx)
		if pendingErr != nil {
			return nil, pendingErr
		}
		return &hydrav1.EvaluateDeployAnomalyResponse{Pending: pending}, nil
	}

	if err := h.reconcile(ctx, req); err != nil {
		return nil, err
	}

	cfg := DefaultConfig(SensitivityNormal)
	processed := false
	for _, metricValue := range req.GetMetrics() {
		metric := Metric(metricValue.GetMetric())
		if !validMetric(metric) {
			return nil, restate.TerminalError(fault.New(fmt.Sprintf("unsupported deploy anomaly metric %q", metric)))
		}

		if metricValue.GetDataState() == hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE {
			logger.Warn("deploy anomaly metric skipped because ingest is incomplete",
				"workspace_id", req.GetWorkspaceId(), "app_id", req.GetAppId(),
				"environment_id", req.GetEnvironmentId(), "metric", metric,
				"window_start", req.GetWindowStart(),
			)
			continue
		}
		processed = true

		openID, err := restate.Get[string](ctx, openAlertKey(metric))
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("get open alert for %s", metric)))
		}
		if metric == MetricRequestsDrop && requestDropSuppressed(req) {
			if err := h.suppressRequestDrop(ctx, req, openID); err != nil {
				return nil, err
			}
			continue
		}

		candidate, err := restate.Get[bool](ctx, candidateKey(metric))
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("get candidate for %s", metric)))
		}
		candidateWindow, err := restate.Get[int64](ctx, candidateWindowKey(metric))
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal(fmt.Sprintf("get candidate window for %s", metric)))
		}
		previousCandidate := candidate && candidateWindow == req.GetWindowStart()-windowDurationMillis

		input := detectorInput(metricValue, req.GetWindowStart(), previousCandidate)
		if metric == MetricRequestsDrop && metricValue.GetDataState() == hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE && (candidate || openID != "") {
			stored, getErr := restate.Get[Input](ctx, detectorInputKey(metric))
			if getErr != nil {
				return nil, fault.Wrap(getErr, fault.Internal(fmt.Sprintf("get stored detector input for %s", metric)))
			}
			stored.Current = 0
			stored.RequestsInWindow = 0
			stored.PreviousCandidate = previousCandidate
			input = stored
		}

		if openID != "" {
			if err := h.evaluateOpen(ctx, req, input, openID, cfg); err != nil {
				return nil, err
			}
			continue
		}

		result := Detect(input, cfg)
		switch result.Outcome {
		case OutcomeCandidate:
			restate.Set(ctx, candidateKey(metric), true)
			restate.Set(ctx, candidateWindowKey(metric), req.GetWindowStart())
			restate.Set(ctx, detectorInputKey(metric), input)
		case OutcomeAnomaly:
			restate.Clear(ctx, candidateKey(metric))
			restate.Clear(ctx, candidateWindowKey(metric))
			if err := h.open(ctx, req, input, result); err != nil {
				return nil, err
			}
		case OutcomeNone, OutcomeInsufficient:
			restate.Clear(ctx, candidateKey(metric))
			restate.Clear(ctx, candidateWindowKey(metric))
			restate.Clear(ctx, detectorInputKey(metric))
		default:
			return nil, restate.TerminalError(fault.New(fmt.Sprintf("unsupported detector outcome %q", result.Outcome)))
		}
	}
	if processed {
		restate.Set(ctx, lastWindowEndStateKey, req.GetWindowEnd())
	}

	pending, err := hasPendingState(ctx)
	if err != nil {
		return nil, err
	}
	return &hydrav1.EvaluateDeployAnomalyResponse{Pending: pending}, nil
}

func (h *CheckHandler) reconcile(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest) error {
	reconciled, err := restate.Get[bool](ctx, "open_alerts_reconciled")
	if err != nil {
		return fault.Wrap(err, fault.Internal("get reconciliation state"))
	}
	if reconciled {
		return nil
	}
	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.AlertEvent, error) {
		return h.db.FindOpenAlertEventsByGroup(rc, db.FindOpenAlertEventsByGroupParams{
			WorkspaceID: req.GetWorkspaceId(), AppID: req.GetAppId(), EnvironmentID: req.GetEnvironmentId(),
		})
	}, restate.WithName("reconcile open anomaly alerts"))
	if err != nil {
		return fault.Wrap(err, fault.Internal("reconcile open anomaly alerts"))
	}
	cfg := DefaultConfig(SensitivityNormal)
	for _, row := range rows {
		metric := Metric(row.Metric)
		if !validMetric(metric) {
			continue
		}
		snapshot := Result{
			Outcome: OutcomeAnomaly, Observed: row.ObservedValue,
			BaselineMean: row.BaselineMean, BaselineStddev: row.BaselineStddev,
			ThresholdValue: openingThreshold(metric, row.BaselineMean, row.BaselineStddev, row.ThresholdSigma, cfg),
			SigmaK:         row.ThresholdSigma, RawCount: 0, Requests: 0, ExpectedCount: 0,
			Catastrophic: false,
			Reason:       "reconciled from open alert",
		}
		restate.Set(ctx, openAlertKey(metric), row.ID)
		restate.Set(ctx, firedAtKey(metric), row.FiredAt)
		restate.Set(ctx, snapshotKey(metric), snapshot)
	}
	restate.Set(ctx, "open_alerts_reconciled", true)
	return nil
}

func openingThreshold(metric Metric, mean, stddev, sigma float64, cfg Config) float64 {
	switch metric {
	case MetricRequestsDrop:
		return mean * cfg.RequestDrop.RecentLevelFraction
	case MetricMemoryUtilization:
		return cfg.ActivityFloors.MemoryUtilization
	case MetricOOMKilled:
		return cfg.ActivityFloors.OOMKilled
	case MetricCrashLoop:
		return cfg.ActivityFloors.CrashLoop
	case MetricError5xx, MetricError4xx, MetricRequests, MetricEgressBytes, MetricCPUSeconds:
		return mean + sigma*stddev
	default:
		return mean + sigma*stddev
	}
}

type openResult struct {
	ID       string
	FiredAt  int64
	Snapshot Result
}

func (h *CheckHandler) open(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, input Input, result Result) error {
	opened, err := restate.Run(ctx, func(rc restate.RunContext) (openResult, error) {
		existing, findErr := h.db.FindOpenAlertEventsByGroup(rc, db.FindOpenAlertEventsByGroupParams{
			WorkspaceID: req.GetWorkspaceId(), AppID: req.GetAppId(), EnvironmentID: req.GetEnvironmentId(),
		})
		if findErr != nil {
			var empty openResult
			return empty, findErr
		}
		for _, alert := range existing {
			if Metric(alert.Metric) == input.Metric {
				cfg := DefaultConfig(SensitivityNormal)
				return openResult{
					ID: alert.ID, FiredAt: alert.FiredAt,
					Snapshot: Result{
						Outcome: OutcomeAnomaly, Observed: alert.ObservedValue,
						BaselineMean: alert.BaselineMean, BaselineStddev: alert.BaselineStddev,
						ThresholdValue: openingThreshold(input.Metric, alert.BaselineMean, alert.BaselineStddev, alert.ThresholdSigma, cfg),
						SigmaK:         alert.ThresholdSigma, RawCount: 0, Requests: 0, ExpectedCount: 0,
						Catastrophic: false,
						Reason:       "reconciled from open alert",
					},
				}, nil
			}
		}
		id := uid.New(uid.AlertPrefix)
		err := h.db.InsertAlertEvent(rc, db.InsertAlertEventParams{
			ID: id, WorkspaceID: req.GetWorkspaceId(), ProjectID: req.GetProjectId(),
			AppID: req.GetAppId(), EnvironmentID: req.GetEnvironmentId(),
			DeploymentID: sql.NullString{String: req.GetDeploymentId(), Valid: req.GetDeploymentId() != ""},
			Metric:       db.AlertEventsMetric(input.Metric), FiredAt: req.GetWindowEnd(), LastSeenAt: req.GetWindowEnd(),
			ObservedValue: result.Observed, BaselineMean: result.BaselineMean,
			BaselineStddev: result.BaselineStddev, ThresholdSigma: result.SigmaK,
			WindowStart: req.GetWindowStart(), WindowEnd: req.GetWindowEnd(), CreatedAt: req.GetWindowEnd(),
			UpdatedAt: sql.NullInt64{},
		})
		return openResult{ID: id, FiredAt: req.GetWindowEnd(), Snapshot: result}, err
	}, restate.WithName("insert anomaly alert"))
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("insert anomaly alert for %s", input.Metric)))
	}
	alertID := opened.ID

	restate.Set(ctx, openAlertKey(input.Metric), alertID)
	restate.Set(ctx, firedAtKey(input.Metric), opened.FiredAt)
	restate.Set(ctx, snapshotKey(input.Metric), opened.Snapshot)
	restate.Set(ctx, detectorInputKey(input.Metric), input)
	restate.Set(ctx, quietKey(input.Metric), 0)

	logger.Info("deploy anomaly alert opened",
		"alert_id", alertID, "workspace_id", req.GetWorkspaceId(), "app_id", req.GetAppId(),
		"environment_id", req.GetEnvironmentId(), "metric", input.Metric,
		"current", input.Current, "maximum", input.Maximum, "baseline_mean", input.BaselineMean,
		"baseline_stddev", input.BaselineStddev, "observed_buckets", input.ObservedBaselineBuckets,
		"baseline_window_buckets", input.BaselineWindowBuckets, "first_bucket_time", input.FirstBucketTime,
		"requests_current", input.RequestsInWindow, "recent_median_requests", input.RecentMedianRequests,
		"recent_active_buckets", input.RecentActiveBuckets,
		"sigma", result.SigmaK, "threshold", result.ThresholdValue,
	)
	return nil
}

func (h *CheckHandler) evaluateOpen(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, input Input, alertID string, cfg Config) error {
	firedAt, err := restate.Get[int64](ctx, firedAtKey(input.Metric))
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("get fired time for %s", input.Metric)))
	}
	if maxOpenDurationReached(firedAt, req.GetWindowEnd(), cfg.MaxOpenDuration) {
		return h.resolve(ctx, req, alertID, input.Metric, baselineAdaptedMessage)
	}
	snapshot, err := restate.Get[Result](ctx, snapshotKey(input.Metric))
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("get opening snapshot for %s", input.Metric)))
	}
	if !Recovered(input, snapshot, cfg) {
		restate.Set(ctx, quietKey(input.Metric), 0)
		restate.Set(ctx, detectorInputKey(input.Metric), input)
		return h.touch(ctx, alertID, req.GetWindowEnd(), observedValue(input))
	}

	quiet, err := restate.Get[int](ctx, quietKey(input.Metric))
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("get quiet windows for %s", input.Metric)))
	}
	quiet++
	if !ShouldResolve(quiet) {
		restate.Set(ctx, quietKey(input.Metric), quiet)
		return nil
	}
	return h.resolve(ctx, req, alertID, input.Metric, autoResolveMessage)
}

func maxOpenDurationReached(firedAt, windowEnd int64, duration time.Duration) bool {
	return firedAt > 0 && windowEnd >= firedAt && time.Duration(windowEnd-firedAt)*time.Millisecond >= duration
}

func (h *CheckHandler) touch(ctx restate.ObjectContext, alertID string, windowEnd int64, observed float64) error {
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.db.TouchAlertEventLastSeen(rc, db.TouchAlertEventLastSeenParams{
			ID: alertID, LastSeenAt: windowEnd, ObservedValue: observed,
			UpdatedAt: sql.NullInt64{Int64: windowEnd, Valid: true},
		})
	}, restate.WithName("touch anomaly alert"))
}

func (h *CheckHandler) resolve(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, alertID string, metric Metric, message string) error {
	_, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return h.db.ResolveAlertEventBySystem(rc, db.ResolveAlertEventBySystemParams{
			ID:                alertID,
			ResolvedAt:        sql.NullInt64{Int64: req.GetWindowEnd(), Valid: true},
			ResolutionMessage: sql.NullString{String: message, Valid: true},
			UpdatedAt:         sql.NullInt64{Int64: req.GetWindowEnd(), Valid: true},
		})
	}, restate.WithName("resolve anomaly alert"))
	if err != nil {
		return fault.Wrap(err, fault.Internal(fmt.Sprintf("resolve anomaly alert %s", alertID)))
	}
	clearMetricState(ctx, metric)
	return nil
}

func (h *CheckHandler) suppressRequestDrop(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, openID string) error {
	logger.Info("deploy request-drop alert suppressed for inactive deployment",
		"workspace_id", req.GetWorkspaceId(), "app_id", req.GetAppId(),
		"environment_id", req.GetEnvironmentId(), "deployment_id", req.GetDeploymentId(),
		"desired_state", req.GetDeploymentDesiredState(),
		"has_running_region", req.GetDeploymentHasRunningRegion(),
	)
	if openID == "" {
		clearMetricState(ctx, MetricRequestsDrop)
		return nil
	}
	return h.resolve(ctx, req, openID, MetricRequestsDrop, stoppedMessage)
}

func requestDropSuppressed(req *hydrav1.EvaluateDeployAnomalyRequest) bool {
	return req.GetDeploymentId() == "" || req.GetDeploymentDesiredState() == "stopped" || !req.GetDeploymentHasRunningRegion()
}

func observedValue(input Input) float64 {
	if input.Metric == MetricError5xx || input.Metric == MetricError4xx {
		if input.RequestsInWindow == 0 {
			return 0
		}
		return input.Current / input.RequestsInWindow
	}
	return input.Current
}

func clearMetricState(ctx restate.ObjectContext, metric Metric) {
	restate.Clear(ctx, candidateKey(metric))
	restate.Clear(ctx, candidateWindowKey(metric))
	restate.Clear(ctx, openAlertKey(metric))
	restate.Clear(ctx, firedAtKey(metric))
	restate.Clear(ctx, quietKey(metric))
	restate.Clear(ctx, snapshotKey(metric))
	restate.Clear(ctx, detectorInputKey(metric))
}

func hasPendingState(ctx restate.ObjectContext) (bool, error) {
	for _, metric := range allMetrics {
		openID, err := restate.Get[string](ctx, openAlertKey(metric))
		if err != nil {
			return false, fault.Wrap(err, fault.Internal(fmt.Sprintf("get open alert for %s", metric)))
		}
		candidate, err := restate.Get[bool](ctx, candidateKey(metric))
		if err != nil {
			return false, fault.Wrap(err, fault.Internal(fmt.Sprintf("get candidate for %s", metric)))
		}
		if openID != "" || candidate {
			return true, nil
		}
	}
	return false, nil
}

func validMetric(metric Metric) bool {
	for _, candidate := range allMetrics {
		if candidate == metric {
			return true
		}
	}
	return false
}
