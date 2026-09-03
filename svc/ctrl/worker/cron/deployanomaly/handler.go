package deployanomaly

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const windowDurationMillis = int64(5 * 60 * 1_000)

// HandlerConfig holds the fleet anomaly orchestrator dependencies.
type HandlerConfig struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
	Heartbeat  healthcheck.Heartbeat
}

// Handler reads one closed fleet window and dispatches actionable production
// app and environment groups to their stateful evaluator.
type Handler struct {
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
}

// NewHandler constructs the fleet anomaly orchestrator.
func NewHandler(cfg HandlerConfig) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop()"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, clickhouse: cfg.Clickhouse, heartbeat: cfg.Heartbeat}, nil
}

type anomalyGroup struct {
	WorkspaceID   string
	ProjectID     string
	AppID         string
	EnvironmentID string
}

func (g anomalyGroup) key() string {
	return strings.Join([]string{g.WorkspaceID, g.AppID, g.EnvironmentID}, "/")
}

type groupWindow struct {
	request  *clickhouse.RequestAnomalyWindow
	resource *clickhouse.ResourceAnomalyWindow
	events   *clickhouse.InstanceEventAnomalyWindow
	open     bool
}

type groupMetadata struct {
	Group                  anomalyGroup
	OrgID                  string
	WorkspaceName          string
	WorkspaceSlug          string
	AppName                string
	EnvironmentKind        mysqltype.EnvironmentKind
	EnvironmentSlug        string
	DeploymentID           string
	DeploymentDesiredState string
	NotificationsMuted     bool
}

// ParseWindowStart extracts and validates the aligned unix-second window from
// a deploy anomaly cron VO key.
func ParseWindowStart(key string) (int64, error) {
	value := strings.TrimPrefix(key, "deploy-anomaly-")
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 || seconds%300 != 0 {
		return 0, fmt.Errorf("invalid deploy anomaly window key %q", key)
	}
	return seconds * 1_000, nil
}

// Handle evaluates one window encoded in the CronService object key.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeployAnomalyCheckRequest,
) (*hydrav1.RunDeployAnomalyCheckResponse, error) {
	windowStart, err := ParseWindowStart(restate.Key(ctx))
	if err != nil {
		return nil, restate.TerminalError(err)
	}
	windowEnd := windowStart + windowDurationMillis
	req := clickhouse.AnomalyWindowsRequest{WindowStart: windowStart, WorkspaceIDs: nil}

	watermarks, err := restate.Run(ctx, func(rc restate.RunContext) (clickhouse.AnomalySourceWatermarks, error) {
		return h.clickhouse.GetAnomalySourceWatermarks(rc)
	}, restate.WithName("read anomaly ingest watermarks"))
	if err != nil {
		return nil, fmt.Errorf("read anomaly ingest watermarks: %w", err)
	}
	requestWindows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.RequestAnomalyWindow, error) {
		return h.clickhouse.GetRequestAnomalyWindows(rc, req)
	}, restate.WithName("read request anomaly windows"))
	if err != nil {
		return nil, fmt.Errorf("read request anomaly windows: %w", err)
	}
	resourceWindows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ResourceAnomalyWindow, error) {
		return h.clickhouse.GetResourceAnomalyWindows(rc, req)
	}, restate.WithName("read resource anomaly windows"))
	if err != nil {
		return nil, fmt.Errorf("read resource anomaly windows: %w", err)
	}
	eventWindows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceEventAnomalyWindow, error) {
		return h.clickhouse.GetInstanceEventAnomalyWindows(rc, req)
	}, restate.WithName("read instance event anomaly windows"))
	if err != nil {
		return nil, fmt.Errorf("read instance event anomaly windows: %w", err)
	}
	openGroups, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListOpenAlertEventGroupsRow, error) {
		return h.db.ListOpenAlertEventGroups(rc)
	}, restate.WithName("list open anomaly groups"))
	if err != nil {
		return nil, fmt.Errorf("list open anomaly groups: %w", err)
	}

	groups := mergeGroupWindows(requestWindows, resourceWindows, eventWindows, openGroups)
	keys := actionableGroups(groups, windowStart, windowEnd, watermarks)
	metadata, err := restate.Run(ctx, func(rc restate.RunContext) ([]groupMetadata, error) {
		resolved := make([]groupMetadata, 0, len(keys))
		for _, group := range keys {
			row, findErr := h.db.FindLiveDeploymentForEnvironment(rc, db.FindLiveDeploymentForEnvironmentParams{
				WorkspaceID: group.WorkspaceID, ProjectID: group.ProjectID,
				AppID: group.AppID, EnvironmentID: group.EnvironmentID,
			})
			if findErr != nil {
				if db.IsNotFound(findErr) {
					continue
				}
				return nil, findErr
			}
			resolved = append(resolved, groupMetadata{
				Group: group, OrgID: row.OrgID, WorkspaceName: row.WorkspaceName,
				WorkspaceSlug: row.WorkspaceSlug, AppName: row.AppName,
				EnvironmentKind: row.EnvironmentKind, EnvironmentSlug: row.EnvironmentSlug,
				DeploymentID: row.DeploymentID.String, DeploymentDesiredState: string(row.DeploymentDesiredState),
				NotificationsMuted: row.NotificationsMuted != 0,
			})
		}
		return resolved, nil
	}, restate.WithName("resolve anomaly group metadata"))
	if err != nil {
		return nil, fmt.Errorf("resolve anomaly group metadata: %w", err)
	}

	type evaluateFuture = restate.ResponseFuture[*hydrav1.EvaluateDeployAnomalyResponse]
	futures := make([]evaluateFuture, 0, len(metadata))
	dispatchedGroups := make([]anomalyGroup, 0, len(metadata))
	for _, item := range metadata {
		if !isProduction(item.EnvironmentKind) {
			continue
		}
		group := groups[item.Group]
		request := evaluateRequest(item, group, windowStart, windowEnd, watermarks)
		futures = append(futures, hydrav1.NewDeployAnomalyServiceClient(ctx, item.Group.key()).
			Evaluate().RequestFuture(request))
		dispatchedGroups = append(dispatchedGroups, item.Group)
	}

	failed := 0
	for i, future := range futures {
		if _, responseErr := future.Response(); responseErr != nil {
			failed++
			logger.Error("deploy anomaly child failed", "group", dispatchedGroups[i].key(), "error", responseErr)
		}
	}
	if failed == 0 {
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return h.heartbeat.Ping(rc)
		}, restate.WithName("send heartbeat")); err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
	} else {
		logger.Error("deploy anomaly check withheld heartbeat after child failures", "failed", failed)
	}

	return &hydrav1.RunDeployAnomalyCheckResponse{GroupsDispatched: int32(len(futures))}, nil
}

func mergeGroupWindows(
	request []clickhouse.RequestAnomalyWindow,
	resource []clickhouse.ResourceAnomalyWindow,
	events []clickhouse.InstanceEventAnomalyWindow,
	open []db.ListOpenAlertEventGroupsRow,
) map[anomalyGroup]groupWindow {
	groups := make(map[anomalyGroup]groupWindow, len(request)+len(resource)+len(events)+len(open))
	for i := range request {
		group := anomalyGroup{request[i].WorkspaceID, request[i].ProjectID, request[i].AppID, request[i].EnvironmentID}
		value := groups[group]
		value.request = &request[i]
		groups[group] = value
	}
	for i := range resource {
		group := anomalyGroup{resource[i].WorkspaceID, resource[i].ProjectID, resource[i].AppID, resource[i].EnvironmentID}
		value := groups[group]
		value.resource = &resource[i]
		groups[group] = value
	}
	for i := range events {
		group := anomalyGroup{events[i].WorkspaceID, events[i].ProjectID, events[i].AppID, events[i].EnvironmentID}
		value := groups[group]
		value.events = &events[i]
		groups[group] = value
	}
	for _, row := range open {
		group := anomalyGroup{row.WorkspaceID, row.ProjectID, row.AppID, row.EnvironmentID}
		value := groups[group]
		value.open = true
		groups[group] = value
	}
	return groups
}

func actionableGroups(groups map[anomalyGroup]groupWindow, windowStart, windowEnd int64, watermarks clickhouse.AnomalySourceWatermarks) []anomalyGroup {
	cfg := DefaultConfig(SensitivityNormal)
	actionable := make([]anomalyGroup, 0, len(groups))
	for group, values := range groups {
		if values.open || hasCandidate(evaluateMetrics(values, windowStart, windowEnd, watermarks), windowStart, cfg) {
			actionable = append(actionable, group)
		}
	}
	sort.Slice(actionable, func(i, j int) bool {
		if actionable[i].WorkspaceID != actionable[j].WorkspaceID {
			return actionable[i].WorkspaceID < actionable[j].WorkspaceID
		}
		if actionable[i].ProjectID != actionable[j].ProjectID {
			return actionable[i].ProjectID < actionable[j].ProjectID
		}
		if actionable[i].AppID != actionable[j].AppID {
			return actionable[i].AppID < actionable[j].AppID
		}
		return actionable[i].EnvironmentID < actionable[j].EnvironmentID
	})
	return actionable
}

func hasCandidate(metrics []*hydrav1.DeployAnomalyMetricInput, windowStart int64, cfg Config) bool {
	for _, metric := range metrics {
		if metric.GetDataState() == hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE {
			continue
		}
		result := Detect(detectorInput(metric, windowStart, false), cfg)
		if result.Outcome == OutcomeCandidate || result.Outcome == OutcomeAnomaly {
			return true
		}
	}
	return false
}

func isProduction(kind mysqltype.EnvironmentKind) bool {
	return kind == mysqltype.EnvironmentKindProduction
}

func evaluateRequest(metadata groupMetadata, group groupWindow, windowStart, windowEnd int64, watermarks clickhouse.AnomalySourceWatermarks) *hydrav1.EvaluateDeployAnomalyRequest {
	return &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowStart, WindowEnd: windowEnd,
		WorkspaceId: metadata.Group.WorkspaceID, ProjectId: metadata.Group.ProjectID,
		AppId: metadata.Group.AppID, EnvironmentId: metadata.Group.EnvironmentID,
		OrgId: metadata.OrgID, WorkspaceName: metadata.WorkspaceName,
		WorkspaceSlug: metadata.WorkspaceSlug, AppName: metadata.AppName,
		EnvironmentSlug: metadata.EnvironmentSlug, DeploymentId: metadata.DeploymentID,
		DeploymentDesiredState: metadata.DeploymentDesiredState,
		Metrics:                evaluateMetrics(group, windowStart, windowEnd, watermarks),
		NotificationsMuted:     metadata.NotificationsMuted,
	}
}

func evaluateMetrics(group groupWindow, windowStart, windowEnd int64, watermarks clickhouse.AnomalySourceWatermarks) []*hydrav1.DeployAnomalyMetricInput {
	requestPresent := group.request != nil && group.request.CurrentBucketPresent
	requestState := metricDataState(requestPresent, watermarks.Requests >= windowEnd)
	resourceState := metricDataState(group.resource != nil, watermarks.Resources >= windowEnd)
	eventState := hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE
	if group.events != nil {
		eventState = hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT
	}

	metrics := make([]*hydrav1.DeployAnomalyMetricInput, 0, 9)
	if row := group.request; row != nil {
		median, active := RecentRequestStats(row.RecentRequests, DefaultConfig(SensitivityNormal).RequestDrop.ActivityPerBucket)
		metrics = append(metrics,
			metricInput(MetricError5xx, requestState, row.Error5xxCurrent, row.Error5xxBaselineMean, row.Error5xxBaselineStddev, row.BaselineBuckets, row.FirstBucketTime, row.RequestsCurrent),
			metricInput(MetricError4xx, requestState, row.Error4xxCurrent, row.Error4xxBaselineMean, row.Error4xxBaselineStddev, row.BaselineBuckets, row.FirstBucketTime, row.RequestsCurrent),
			metricInput(MetricRequests, requestState, row.RequestsCurrent, row.RequestsBaselineMean, row.RequestsBaselineStddev, row.BaselineBuckets, row.FirstBucketTime, row.RequestsCurrent),
			metricInput(MetricRequestsDrop, requestState, row.RequestsCurrent, row.RequestsBaselineMean, row.RequestsBaselineStddev, row.BaselineBuckets, row.FirstBucketTime, row.RequestsCurrent),
		)
		metrics[len(metrics)-1].RecentMedianRequests = median
		metrics[len(metrics)-1].RecentActiveBuckets = active
	} else {
		for _, metric := range []Metric{MetricError5xx, MetricError4xx, MetricRequests, MetricRequestsDrop} {
			metrics = append(metrics, metricInput(metric, requestState, 0, 0, 0, 0, 0, 0))
		}
	}

	if row := group.resource; row != nil {
		metrics = append(metrics,
			metricInput(MetricEgressBytes, resourceState, row.EgressBytesCurrent, row.EgressBytesBaselineMean, row.EgressBytesBaselineStddev, row.BaselineBuckets, row.FirstBucketTime, 0),
			metricInput(MetricCPUSeconds, resourceState, row.CPUSecondsCurrent, row.CPUSecondsBaselineMean, row.CPUSecondsBaselineStddev, row.BaselineBuckets, row.FirstBucketTime, 0),
			metricInput(MetricMemoryUtilization, resourceState, row.MemoryUtilizationCurrent, 0, 0, 0, 0, 0),
		)
		metrics[len(metrics)-1].Maximum = row.MemoryUtilizationMaxCurrent
	} else {
		for _, metric := range []Metric{MetricEgressBytes, MetricCPUSeconds, MetricMemoryUtilization} {
			metrics = append(metrics, metricInput(metric, resourceState, 0, 0, 0, 0, 0, 0))
		}
	}

	if row := group.events; row != nil {
		metrics = append(metrics,
			metricInput(MetricOOMKilled, eventState, row.OOMKilledCurrent, 0, 0, 0, 0, 0),
			metricInput(MetricCrashLoop, eventState, row.CrashLoopCurrent, 0, 0, 0, 0, 0),
		)
	} else {
		metrics = append(metrics,
			metricInput(MetricOOMKilled, eventState, 0, 0, 0, 0, 0, 0),
			metricInput(MetricCrashLoop, eventState, 0, 0, 0, 0, 0, 0),
		)
	}
	return metrics
}

func metricDataState(present, complete bool) hydrav1.DeployAnomalyMetricDataState {
	if !complete {
		return hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_INCOMPLETE
	}
	if present {
		return hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_PRESENT
	}
	return hydrav1.DeployAnomalyMetricDataState_DEPLOY_ANOMALY_METRIC_DATA_STATE_ZERO_COMPLETE
}

func metricInput(metric Metric, state hydrav1.DeployAnomalyMetricDataState, current, mean, stddev float64, buckets, firstBucket int64, requests float64) *hydrav1.DeployAnomalyMetricInput {
	return &hydrav1.DeployAnomalyMetricInput{
		Metric: string(metric), DataState: state, Current: current,
		BaselineMean: mean, BaselineStddev: stddev,
		ObservedBaselineBuckets: buckets, FirstBucketTime: firstBucket,
		RequestsInWindow: requests,
	}
}

func detectorInput(metric *hydrav1.DeployAnomalyMetricInput, windowStart int64, previousCandidate bool) Input {
	return Input{
		Metric: Metric(metric.GetMetric()), Current: metric.GetCurrent(), Maximum: metric.GetMaximum(),
		RequestsInWindow:     metric.GetRequestsInWindow(),
		RecentMedianRequests: metric.GetRecentMedianRequests(),
		RecentActiveBuckets:  metric.GetRecentActiveBuckets(),
		BaselineMean:         metric.GetBaselineMean(), BaselineStddev: metric.GetBaselineStddev(),
		ObservedBaselineBuckets: metric.GetObservedBaselineBuckets(),
		BaselineWindowBuckets:   BaselineWindowBuckets(windowStart, metric.GetFirstBucketTime()),
		FirstBucketTime:         metric.GetFirstBucketTime(),
		PreviousCandidate:       previousCandidate,
	}
}
