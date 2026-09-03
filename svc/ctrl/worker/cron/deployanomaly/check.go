package deployanomaly

import (
	"database/sql"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

const (
	autoResolveMessage = "Metric returned to baseline for 3 consecutive windows"
	stoppedMessage     = "Deployment stopped"
)

// CheckConfig holds the per-group evaluator dependencies.
type CheckConfig struct {
	DB               db.Database
	Admins           workos.Resolver
	Email            email.Sender
	DashboardBaseURL string
}

// CheckHandler owns durable anomaly state for one app and environment.
type CheckHandler struct {
	db               db.Database
	admins           workos.Resolver
	email            email.Sender
	dashboardBaseURL string
}

// NewCheckHandler constructs the per-group anomaly evaluator.
func NewCheckHandler(cfg CheckConfig) (*CheckHandler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Admins, "Admins must not be nil; use workos.NewNoop()"),
		assert.NotNil(cfg.Email, "Email must not be nil; use email.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &CheckHandler{db: cfg.DB, admins: cfg.Admins, email: cfg.Email, dashboardBaseURL: cfg.DashboardBaseURL}, nil
}

func candidateKey(metric Metric) string       { return "candidate:" + string(metric) }
func candidateWindowKey(metric Metric) string { return "candidate_window:" + string(metric) }
func openAlertKey(metric Metric) string       { return "open_alert:" + string(metric) }
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
	if err := h.reconcile(ctx, req); err != nil {
		return nil, err
	}

	cfg := DefaultConfig(SensitivityNormal)
	for _, metricValue := range req.GetMetrics() {
		metric := Metric(metricValue.GetMetric())
		if !validMetric(metric) {
			return nil, restate.TerminalError(fmt.Errorf("unsupported deploy anomaly metric %q", metric))
		}

		if metricValue.GetDataState() == hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE {
			logger.Warn("deploy anomaly metric skipped because ingest is incomplete",
				"workspace_id", req.GetWorkspaceId(), "app_id", req.GetAppId(),
				"environment_id", req.GetEnvironmentId(), "metric", metric,
				"window_start", req.GetWindowStart(),
			)
			continue
		}

		openID, err := restate.Get[string](ctx, openAlertKey(metric))
		if err != nil {
			return nil, fmt.Errorf("get open alert for %s: %w", metric, err)
		}
		if metric == MetricRequestsDrop && requestDropSuppressed(req) {
			if err := h.suppressRequestDrop(ctx, req, openID); err != nil {
				return nil, err
			}
			continue
		}

		candidate, err := restate.Get[bool](ctx, candidateKey(metric))
		if err != nil {
			return nil, fmt.Errorf("get candidate for %s: %w", metric, err)
		}
		candidateWindow, err := restate.Get[int64](ctx, candidateWindowKey(metric))
		if err != nil {
			return nil, fmt.Errorf("get candidate window for %s: %w", metric, err)
		}
		previousCandidate := candidate && candidateWindow == req.GetWindowStart()-windowDurationMillis

		input := detectorInput(metricValue, req.GetWindowStart(), previousCandidate)
		if metric == MetricRequestsDrop && metricValue.GetDataState() == hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE && (candidate || openID != "") {
			stored, getErr := restate.Get[Input](ctx, detectorInputKey(metric))
			if getErr != nil {
				return nil, fmt.Errorf("get stored detector input for %s: %w", metric, getErr)
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
			return nil, restate.TerminalError(fmt.Errorf("unsupported detector outcome %q", result.Outcome))
		}
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
		return fmt.Errorf("get reconciliation state: %w", err)
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
		return fmt.Errorf("reconcile open anomaly alerts: %w", err)
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
			Notify: shouldNotify(metric, cfg.Notifications), Catastrophic: false,
			Reason: "reconciled from open alert",
		}
		restate.Set(ctx, openAlertKey(metric), row.ID)
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
	Created  bool
	Snapshot Result
}

func (h *CheckHandler) open(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, input Input, result Result) error {
	opened, err := restate.Run(ctx, func(rc restate.RunContext) (openResult, error) {
		existing, findErr := h.db.FindOpenAlertEventsByGroup(rc, db.FindOpenAlertEventsByGroupParams{
			WorkspaceID: req.GetWorkspaceId(), AppID: req.GetAppId(), EnvironmentID: req.GetEnvironmentId(),
		})
		if findErr != nil {
			return openResult{}, findErr
		}
		for _, alert := range existing {
			if Metric(alert.Metric) == input.Metric {
				cfg := DefaultConfig(SensitivityNormal)
				return openResult{
					ID: alert.ID, Created: false,
					Snapshot: Result{
						Outcome: OutcomeAnomaly, Observed: alert.ObservedValue,
						BaselineMean: alert.BaselineMean, BaselineStddev: alert.BaselineStddev,
						ThresholdValue: openingThreshold(input.Metric, alert.BaselineMean, alert.BaselineStddev, alert.ThresholdSigma, cfg),
						SigmaK:         alert.ThresholdSigma, RawCount: 0, Requests: 0, ExpectedCount: 0,
						Notify: shouldNotify(input.Metric, cfg.Notifications), Catastrophic: false,
						Reason: "reconciled from open alert",
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
		return openResult{ID: id, Created: true, Snapshot: result}, err
	}, restate.WithName("insert anomaly alert"))
	if err != nil {
		return fmt.Errorf("insert anomaly alert for %s: %w", input.Metric, err)
	}
	alertID := opened.ID

	restate.Set(ctx, openAlertKey(input.Metric), alertID)
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
	if opened.Created && result.Notify && !req.GetNotificationsMuted() {
		if err := h.sendAlert(ctx, req, alertID, input.Metric, result); err != nil {
			return err
		}
	} else if opened.Created {
		logger.Info("deploy anomaly email suppressed", "alert_id", alertID, "metric", input.Metric,
			"workspace_muted", req.GetNotificationsMuted(), "metric_notify", result.Notify)
	}
	return nil
}

func (h *CheckHandler) evaluateOpen(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, input Input, alertID string, cfg Config) error {
	snapshot, err := restate.Get[Result](ctx, snapshotKey(input.Metric))
	if err != nil {
		return fmt.Errorf("get opening snapshot for %s: %w", input.Metric, err)
	}
	if !Recovered(input, snapshot, cfg) {
		restate.Set(ctx, quietKey(input.Metric), 0)
		restate.Set(ctx, detectorInputKey(input.Metric), input)
		return h.touch(ctx, alertID, req.GetWindowEnd(), observedValue(input))
	}

	quiet, err := restate.Get[int](ctx, quietKey(input.Metric))
	if err != nil {
		return fmt.Errorf("get quiet windows for %s: %w", input.Metric, err)
	}
	quiet++
	if !ShouldResolve(quiet) {
		restate.Set(ctx, quietKey(input.Metric), quiet)
		return nil
	}
	return h.resolve(ctx, req, alertID, input.Metric, snapshot, autoResolveMessage)
}

func (h *CheckHandler) touch(ctx restate.ObjectContext, alertID string, windowEnd int64, observed float64) error {
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.db.TouchAlertEventLastSeen(rc, db.TouchAlertEventLastSeenParams{
			ID: alertID, LastSeenAt: windowEnd, ObservedValue: observed,
			UpdatedAt: sql.NullInt64{Int64: windowEnd, Valid: true},
		})
	}, restate.WithName("touch anomaly alert"))
}

func (h *CheckHandler) resolve(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, alertID string, metric Metric, snapshot Result, message string) error {
	rows, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
		return h.db.ResolveAlertEventBySystem(rc, db.ResolveAlertEventBySystemParams{
			ID:                alertID,
			ResolvedAt:        sql.NullInt64{Int64: req.GetWindowEnd(), Valid: true},
			ResolutionMessage: sql.NullString{String: message, Valid: true},
			UpdatedAt:         sql.NullInt64{Int64: req.GetWindowEnd(), Valid: true},
		})
	}, restate.WithName("resolve anomaly alert"))
	if err != nil {
		return fmt.Errorf("resolve anomaly alert %s: %w", alertID, err)
	}
	if rows > 0 && !req.GetNotificationsMuted() && snapshot.Notify {
		if err := h.sendResolved(ctx, req, alertID, metric, snapshot); err != nil {
			return err
		}
	}
	clearMetricState(ctx, metric)
	return nil
}

func (h *CheckHandler) suppressRequestDrop(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, openID string) error {
	logger.Info("deploy request-drop alert suppressed for inactive deployment",
		"workspace_id", req.GetWorkspaceId(), "app_id", req.GetAppId(),
		"environment_id", req.GetEnvironmentId(), "deployment_id", req.GetDeploymentId(),
		"desired_state", req.GetDeploymentDesiredState(),
	)
	if openID == "" {
		clearMetricState(ctx, MetricRequestsDrop)
		return nil
	}
	snapshot, err := restate.Get[Result](ctx, snapshotKey(MetricRequestsDrop))
	if err != nil {
		return fmt.Errorf("get request-drop snapshot: %w", err)
	}
	return h.resolve(ctx, req, openID, MetricRequestsDrop, snapshot, stoppedMessage)
}

func requestDropSuppressed(req *hydrav1.EvaluateDeployAnomalyRequest) bool {
	return req.GetDeploymentId() == "" || req.GetDeploymentDesiredState() == "stopped"
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
	restate.Clear(ctx, quietKey(metric))
	restate.Clear(ctx, snapshotKey(metric))
	restate.Clear(ctx, detectorInputKey(metric))
}

func hasPendingState(ctx restate.ObjectContext) (bool, error) {
	for _, metric := range allMetrics {
		openID, err := restate.Get[string](ctx, openAlertKey(metric))
		if err != nil {
			return false, fmt.Errorf("get open alert for %s: %w", metric, err)
		}
		candidate, err := restate.Get[bool](ctx, candidateKey(metric))
		if err != nil {
			return false, fmt.Errorf("get candidate for %s: %w", metric, err)
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
