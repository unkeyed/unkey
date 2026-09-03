package deployanomaly

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	pendingGroupsStateKey = "pending_groups"
	metadataBatchSize     = 500
)

type anomalyGroup struct {
	WorkspaceID   string `json:"workspace_id"`
	ProjectID     string `json:"project_id"`
	AppID         string `json:"app_id"`
	EnvironmentID string `json:"environment_id"`
}

func (g anomalyGroup) key() string {
	return strings.Join([]string{g.WorkspaceID, g.AppID, g.EnvironmentID}, "/")
}

func (g anomalyGroup) clickhouseKey() clickhouse.AnomalyGroupKey {
	return clickhouse.AnomalyGroupKey{
		WorkspaceID: g.WorkspaceID, ProjectID: g.ProjectID,
		AppID: g.AppID, EnvironmentID: g.EnvironmentID,
	}
}

type groupWindow struct {
	request  *clickhouse.RequestAnomalyWindow
	resource *clickhouse.ResourceAnomalyWindow
	events   *clickhouse.InstanceEventAnomalyWindow
	forced   bool
}

type groupMetadata struct {
	Group                      anomalyGroup
	OrgID                      string
	WorkspaceName              string
	WorkspaceSlug              string
	AppName                    string
	EnvironmentKind            mysqltype.EnvironmentKind
	EnvironmentSlug            string
	DeploymentID               string
	DeploymentDesiredState     string
	DeploymentHasRunningRegion bool
}

// ShardConfig holds the partition evaluator dependencies.
type ShardConfig struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
}

// ShardHandler queries one workspace hash partition and dispatches its
// actionable groups.
type ShardHandler struct {
	hydrav1.UnimplementedDeployAnomalyShardServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
}

// NewShardHandler constructs the partition evaluator.
func NewShardHandler(cfg ShardConfig) (*ShardHandler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &ShardHandler{
		UnimplementedDeployAnomalyShardServiceServer: hydrav1.UnimplementedDeployAnomalyShardServiceServer{},
		db: cfg.DB, clickhouse: cfg.Clickhouse,
	}, nil
}

// EvaluateShard reads only SQL candidates and explicit open or prior-pending
// groups. Every Restate client call remains outside journaled Run closures.
func (h *ShardHandler) EvaluateShard(
	ctx restate.ObjectContext,
	_ *hydrav1.EvaluateDeployAnomalyShardRequest,
) (*hydrav1.EvaluateDeployAnomalyShardResponse, error) {
	windowStart, shard, shardCount, err := ParseShardKey(restate.Key(ctx))
	if err != nil {
		return nil, restate.TerminalError(err)
	}
	windowEnd := windowStart + windowDurationMillis

	previous, err := hydrav1.NewDeployAnomalyShardServiceClient(
		ctx,
		ShardKey(windowStart-windowDurationMillis, shard, shardCount),
	).GetPending().Request(&hydrav1.GetPendingDeployAnomalyGroupsRequest{})
	if err != nil {
		return nil, fmt.Errorf("read previous pending anomaly groups: %w", err)
	}
	openGroups, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListOpenAlertEventGroupsRow, error) {
		return h.db.ListOpenAlertEventGroups(rc, db.ListOpenAlertEventGroupsParams{
			ShardCount: shardCount,
			Shard:      shard,
		})
	}, restate.WithName("list open anomaly groups for shard"))
	if err != nil {
		return nil, fmt.Errorf("list open anomaly groups for shard: %w", err)
	}

	forced := mergeForcedGroups(previous.GetGroups(), openGroups)
	groupKeys := make([]clickhouse.AnomalyGroupKey, len(forced))
	for i := range forced {
		groupKeys[i] = forced[i].clickhouseKey()
	}
	watermarks, err := restate.Run(ctx, func(rc restate.RunContext) (clickhouse.AnomalySourceWatermarks, error) {
		return h.clickhouse.GetAnomalySourceWatermarks(rc)
	}, restate.WithName("read anomaly ingest watermarks"))
	if err != nil {
		return nil, fmt.Errorf("read anomaly ingest watermarks: %w", err)
	}
	completeness := sourceCompleteness(watermarks, windowEnd)
	logIncompleteSources(shard, windowEnd, completeness)

	filter := candidateFilter(DefaultConfig(SensitivityNormal))
	baseRequest := clickhouse.AnomalyWindowsRequest{
		WindowStart: windowStart, WorkspaceIDs: nil, GroupKeys: groupKeys,
		Shard: shard, ShardCount: shardCount, SkipFleet: false, CandidateFilter: &filter,
	}
	requestQuery := baseRequest
	requestQuery.SkipFleet = !completeness.Requests.Complete
	requestWindows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.RequestAnomalyWindow, error) {
		return h.clickhouse.GetRequestAnomalyWindows(rc, requestQuery)
	}, restate.WithName("read request anomaly candidates"))
	if err != nil {
		return nil, fmt.Errorf("read request anomaly candidates: %w", err)
	}
	resourceQuery := baseRequest
	resourceQuery.SkipFleet = !completeness.Resources.Complete
	resourceWindows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ResourceAnomalyWindow, error) {
		return h.clickhouse.GetResourceAnomalyWindows(rc, resourceQuery)
	}, restate.WithName("read resource anomaly candidates"))
	if err != nil {
		return nil, fmt.Errorf("read resource anomaly candidates: %w", err)
	}
	eventQuery := baseRequest
	eventQuery.SkipFleet = !completeness.InstanceEvents.Complete
	eventWindows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceEventAnomalyWindow, error) {
		return h.clickhouse.GetInstanceEventAnomalyWindows(rc, eventQuery)
	}, restate.WithName("read instance event anomaly candidates"))
	if err != nil {
		return nil, fmt.Errorf("read instance event anomaly candidates: %w", err)
	}

	groups := mergeGroupWindows(requestWindows, resourceWindows, eventWindows, forced)
	keys := actionableGroups(groups, windowStart, completeness)
	metadata, err := h.resolveMetadata(ctx, keys)
	if err != nil {
		return nil, err
	}

	type evaluateFuture = restate.ResponseFuture[*hydrav1.EvaluateDeployAnomalyResponse]
	futures := make([]evaluateFuture, 0, len(metadata))
	dispatchedGroups := make([]anomalyGroup, 0, len(metadata))
	for _, item := range metadata {
		if !isProduction(item.EnvironmentKind) {
			continue
		}
		request := evaluateRequest(item, groups[item.Group], windowStart, windowEnd, completeness)
		futures = append(futures, hydrav1.NewDeployAnomalyServiceClient(ctx, item.Group.key()).
			Evaluate().RequestFuture(request))
		dispatchedGroups = append(dispatchedGroups, item.Group)
	}

	pending := make([]anomalyGroup, 0, len(futures))
	for i, future := range futures {
		response, responseErr := future.Response()
		if responseErr != nil {
			return nil, fmt.Errorf("evaluate deploy anomaly group %s: %w", dispatchedGroups[i].key(), responseErr)
		}
		if response.GetPending() {
			pending = append(pending, dispatchedGroups[i])
		}
	}
	sortGroups(pending)
	restate.Set(ctx, pendingGroupsStateKey, pending)
	return &hydrav1.EvaluateDeployAnomalyShardResponse{
		GroupsDispatched: int32(len(futures)),
		GroupsPending:    int32(len(pending)),
	}, nil
}

// GetPending exposes the previous window's small candidate and open-alert set
// without serializing behind its completed exclusive evaluation.
func (h *ShardHandler) GetPending(
	ctx restate.ObjectSharedContext,
	_ *hydrav1.GetPendingDeployAnomalyGroupsRequest,
) (*hydrav1.GetPendingDeployAnomalyGroupsResponse, error) {
	groups, err := restate.Get[[]anomalyGroup](ctx, pendingGroupsStateKey)
	if err != nil {
		return nil, fmt.Errorf("get pending anomaly groups: %w", err)
	}
	response := &hydrav1.GetPendingDeployAnomalyGroupsResponse{Groups: make([]*hydrav1.DeployAnomalyGroupKey, len(groups))}
	for i, group := range groups {
		response.Groups[i] = &hydrav1.DeployAnomalyGroupKey{
			WorkspaceId: group.WorkspaceID, ProjectId: group.ProjectID,
			AppId: group.AppID, EnvironmentId: group.EnvironmentID,
		}
	}
	return response, nil
}

func (h *ShardHandler) resolveMetadata(ctx restate.ObjectContext, groups []anomalyGroup) ([]groupMetadata, error) {
	metadata := make([]groupMetadata, 0, len(groups))
	for start := 0; start < len(groups); start += metadataBatchSize {
		end := min(start+metadataBatchSize, len(groups))
		batch := groups[start:end]
		rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.FindLiveDeploymentsForEnvironmentsRow, error) {
			encoded, marshalErr := json.Marshal(batch)
			if marshalErr != nil {
				return nil, marshalErr
			}
			return h.db.FindLiveDeploymentsForEnvironments(rc, db.FindLiveDeploymentsForEnvironmentsParams{
				GroupKeysJson: string(encoded),
			})
		}, restate.WithName(fmt.Sprintf("resolve anomaly metadata batch %d", start/metadataBatchSize)))
		if err != nil {
			return nil, fmt.Errorf("resolve anomaly group metadata: %w", err)
		}
		for _, row := range rows {
			metadata = append(metadata, groupMetadata{
				Group: anomalyGroup{
					WorkspaceID: row.WorkspaceID, ProjectID: row.ProjectID,
					AppID: row.AppID, EnvironmentID: row.EnvironmentID,
				},
				OrgID: row.OrgID, WorkspaceName: row.WorkspaceName,
				WorkspaceSlug: row.WorkspaceSlug, AppName: row.AppName,
				EnvironmentKind: row.EnvironmentKind, EnvironmentSlug: row.EnvironmentSlug,
				DeploymentID: row.DeploymentID.String, DeploymentDesiredState: string(row.DeploymentDesiredState),
				DeploymentHasRunningRegion: row.DeploymentHasRunningRegion,
			})
		}
	}
	sort.Slice(metadata, func(i, j int) bool { return groupLess(metadata[i].Group, metadata[j].Group) })
	return metadata, nil
}

func mergeForcedGroups(previous []*hydrav1.DeployAnomalyGroupKey, open []db.ListOpenAlertEventGroupsRow) []anomalyGroup {
	unique := make(map[anomalyGroup]struct{}, len(previous)+len(open))
	for _, group := range previous {
		unique[anomalyGroup{group.GetWorkspaceId(), group.GetProjectId(), group.GetAppId(), group.GetEnvironmentId()}] = struct{}{}
	}
	for _, group := range open {
		unique[anomalyGroup{group.WorkspaceID, group.ProjectID, group.AppID, group.EnvironmentID}] = struct{}{}
	}
	groups := make([]anomalyGroup, 0, len(unique))
	for group := range unique {
		groups = append(groups, group)
	}
	sortGroups(groups)
	return groups
}

func mergeGroupWindows(
	request []clickhouse.RequestAnomalyWindow,
	resource []clickhouse.ResourceAnomalyWindow,
	events []clickhouse.InstanceEventAnomalyWindow,
	forced []anomalyGroup,
) map[anomalyGroup]groupWindow {
	groups := make(map[anomalyGroup]groupWindow, len(request)+len(resource)+len(events)+len(forced))
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
	for _, group := range forced {
		value := groups[group]
		value.forced = true
		groups[group] = value
	}
	return groups
}

func actionableGroups(groups map[anomalyGroup]groupWindow, windowStart int64, completeness ingestCompleteness) []anomalyGroup {
	cfg := DefaultConfig(SensitivityNormal)
	actionable := make([]anomalyGroup, 0, len(groups))
	for group, values := range groups {
		if values.forced || hasCandidate(evaluateMetrics(values, windowStart, completeness), windowStart, cfg) {
			actionable = append(actionable, group)
		}
	}
	sortGroups(actionable)
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

func candidateFilter(cfg Config) clickhouse.AnomalyCandidateFilter {
	return clickhouse.AnomalyCandidateFilter{
		SigmaK: cfg.SigmaK, ErrorRatioStddevFloor: cfg.StddevFloors.ErrorRatio,
		RequestsStddevFloor: cfg.StddevFloors.Requests, EgressBytesStddevFloor: cfg.StddevFloors.EgressBytes,
		CPUSecondsStddevFloor: cfg.StddevFloors.CPUSeconds, ErrorExcessFailures: cfg.ActivityFloors.ErrorExcessFailures,
		RequestsActivity: cfg.ActivityFloors.Requests, EgressBytesActivity: cfg.ActivityFloors.EgressBytes,
		CPUSecondsActivity: cfg.ActivityFloors.CPUSeconds, MemoryUtilizationActivity: cfg.ActivityFloors.MemoryUtilization,
		BaselineMinimum: cfg.BaselineMinimums.Requests, RequestDropBaseline: cfg.BaselineMinimums.RequestsDrop,
		RequestDropFraction: cfg.RequestDrop.RecentLevelFraction, RequestDropActivity: cfg.RequestDrop.ActivityPerBucket,
		RequestDropActiveBuckets: cfg.RequestDrop.MinimumActiveBuckets, RequestDropAbsoluteLoss: cfg.RequestDrop.MinimumAbsoluteLoss,
		Catastrophic5xxRatio: cfg.Catastrophic.Error5xxRatio, Catastrophic5xxFailures: cfg.Catastrophic.Error5xxFailures,
	}
}

func isProduction(kind mysqltype.EnvironmentKind) bool {
	return kind == mysqltype.EnvironmentKindProduction
}

func evaluateRequest(metadata groupMetadata, group groupWindow, windowStart, windowEnd int64, completeness ingestCompleteness) *hydrav1.EvaluateDeployAnomalyRequest {
	return &hydrav1.EvaluateDeployAnomalyRequest{
		WindowStart: windowStart, WindowEnd: windowEnd,
		WorkspaceId: metadata.Group.WorkspaceID, ProjectId: metadata.Group.ProjectID,
		AppId: metadata.Group.AppID, EnvironmentId: metadata.Group.EnvironmentID,
		OrgId: metadata.OrgID, WorkspaceName: metadata.WorkspaceName,
		WorkspaceSlug: metadata.WorkspaceSlug, AppName: metadata.AppName,
		EnvironmentSlug: metadata.EnvironmentSlug, DeploymentId: metadata.DeploymentID,
		DeploymentDesiredState:     metadata.DeploymentDesiredState,
		DeploymentHasRunningRegion: metadata.DeploymentHasRunningRegion,
		Metrics:                    evaluateMetrics(group, windowStart, completeness),
	}
}

func evaluateMetrics(group groupWindow, windowStart int64, completeness ingestCompleteness) []*hydrav1.DeployAnomalyMetricInput {
	requestPresent := group.request != nil && group.request.CurrentBucketPresent
	requestState := metricDataState(requestPresent, completeness.Requests.Complete)
	resourcePresent := group.resource != nil && group.resource.CurrentBucketPresent
	resourceState := metricDataState(resourcePresent, completeness.Resources.Complete)
	eventState := metricDataState(group.events != nil, completeness.InstanceEvents.Complete)

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

type sourceStatus struct {
	Complete      bool
	LaggingRegion string
	Watermark     int64
}

type ingestCompleteness struct {
	Requests       sourceStatus
	Resources      sourceStatus
	InstanceEvents sourceStatus
}

func sourceCompleteness(watermarks clickhouse.AnomalySourceWatermarks, windowEnd int64) ingestCompleteness {
	return ingestCompleteness{
		Requests:       sourceStatusFor(watermarks, clickhouse.AnomalySourceRequests, windowEnd),
		Resources:      sourceStatusFor(watermarks, clickhouse.AnomalySourceResources, windowEnd),
		InstanceEvents: sourceStatusFor(watermarks, clickhouse.AnomalySourceInstanceEvents, windowEnd),
	}
}

func sourceStatusFor(watermarks clickhouse.AnomalySourceWatermarks, source string, windowEnd int64) sourceStatus {
	status := sourceStatus{Complete: false, LaggingRegion: "none-active", Watermark: 0}
	found := false
	for _, watermark := range watermarks {
		if watermark.Source != source {
			continue
		}
		if !found || watermark.Watermark < status.Watermark || (watermark.Watermark == status.Watermark && watermark.Region < status.LaggingRegion) {
			status.LaggingRegion = watermark.Region
			status.Watermark = watermark.Watermark
		}
		found = true
	}
	status.Complete = found && status.Watermark >= windowEnd
	return status
}

func logIncompleteSources(shard uint64, windowEnd int64, completeness ingestCompleteness) {
	if shard != 0 {
		return
	}
	for _, item := range []struct {
		source string
		status sourceStatus
	}{
		{source: clickhouse.AnomalySourceRequests, status: completeness.Requests},
		{source: clickhouse.AnomalySourceResources, status: completeness.Resources},
		{source: clickhouse.AnomalySourceInstanceEvents, status: completeness.InstanceEvents},
	} {
		if item.status.Complete {
			continue
		}
		logger.Warn("deploy anomaly source incomplete in active region",
			"source", item.source, "region", item.status.LaggingRegion,
			"watermark", item.status.Watermark, "window_end", windowEnd,
		)
	}
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

func sortGroups(groups []anomalyGroup) {
	sort.Slice(groups, func(i, j int) bool { return groupLess(groups[i], groups[j]) })
}

func groupLess(a, b anomalyGroup) bool {
	if a.WorkspaceID != b.WorkspaceID {
		return a.WorkspaceID < b.WorkspaceID
	}
	if a.ProjectID != b.ProjectID {
		return a.ProjectID < b.ProjectID
	}
	if a.AppID != b.AppID {
		return a.AppID < b.AppID
	}
	return a.EnvironmentID < b.EnvironmentID
}
