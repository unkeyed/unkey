package deployanomaly

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (h *CheckHandler) OpenObservedEvents(ctx restate.ObjectContext, req *hydrav1.OpenObservedDeployAnomalyEventsRequest) (*hydrav1.OpenObservedDeployAnomalyEventsResponse, error) {
	group := req.GetGroup()
	if group.GetWorkspaceId() == "" || group.GetProjectId() == "" || group.GetAppId() == "" || group.GetEnvironmentId() == "" ||
		restate.Key(ctx) != GroupKey(group.GetWorkspaceId(), group.GetAppId(), group.GetEnvironmentId()) ||
		len(req.GetEventIds()) == 0 || len(req.GetEventIds()) > eventPageSize {
		return nil, restate.TerminalError(fault.New("invalid anomaly event group or batch"))
	}
	if !slices.Contains(h.fastWorkspaces, group.GetWorkspaceId()) {
		return &hydrav1.OpenObservedDeployAnomalyEventsResponse{}, nil
	}
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, err
	}
	restate.Clear(ctx, "open_alerts_reconciled")
	result, err := restate.Run(ctx, func(rc restate.RunContext) (*hydrav1.OpenObservedDeployAnomalyEventsResponse, error) {
		return h.processEvents(rc, req, now.UnixMilli())
	}, restate.WithName("commit observed anomaly events"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("process observed anomaly events"))
	}
	if err := h.reconcile(ctx, &hydrav1.EvaluateDeployAnomalyRequest{
		WorkspaceId: group.GetWorkspaceId(), AppId: group.GetAppId(), EnvironmentId: group.GetEnvironmentId(),
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func (h *CheckHandler) processEvents(ctx context.Context, req *hydrav1.OpenObservedDeployAnomalyEventsRequest, now int64) (*hydrav1.OpenObservedDeployAnomalyEventsResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return db.TxWithResult(ctx, h.db.RW(), func(ctx context.Context, tx db.DBTX) (*hydrav1.OpenObservedDeployAnomalyEventsResponse, error) {
		queries := db.NewQueries(tx)
		group := req.GetGroup()
		keys, err := json.Marshal([]anomalyGroup{{
			WorkspaceID: group.GetWorkspaceId(), ProjectID: group.GetProjectId(),
			AppID: group.GetAppId(), EnvironmentID: group.GetEnvironmentId(),
		}})
		if err != nil {
			return nil, err
		}
		metadata, err := queries.FindLiveDeploymentsForEnvironments(ctx, db.FindLiveDeploymentsForEnvironmentsParams{GroupKeysJson: string(keys)})
		if err != nil {
			return nil, err
		}
		deploymentID := ""
		if len(metadata) > 0 && metadata[0].EnvironmentKind == mysqltype.EnvironmentKindProduction &&
			metadata[0].DeploymentDesiredState != "stopped" && metadata[0].DeploymentHasRunningRegion {
			deploymentID = metadata[0].DeploymentID.String
		}
		pending := make([]db.DeployAnomalyEvent, 0, len(req.GetEventIds()))
		ids := slices.Clone(req.GetEventIds())
		slices.Sort(ids)
		for _, id := range slices.Compact(ids) {
			event, err := queries.FindDeployAnomalyEvent(ctx, db.FindDeployAnomalyEventParams{
				ID: id, WorkspaceID: group.GetWorkspaceId(),
			})
			if err != nil {
				return nil, err
			}
			if event.ProjectID != group.GetProjectId() || event.AppID != group.GetAppId() || event.EnvironmentID != group.GetEnvironmentId() {
				return nil, restate.TerminalError(fault.New("anomaly event does not belong to group"))
			}
			if !event.ProcessedAt.Valid {
				pending = append(pending, event)
			}
		}
		response := &hydrav1.OpenObservedDeployAnomalyEventsResponse{}
		for _, metric := range []Metric{MetricOOMKilled, MetricCrashLoop} {
			count := 0
			earliest := now
			for _, event := range pending {
				if deploymentID != "" && event.DeploymentID == deploymentID && Metric(event.Metric) == metric {
					count++
					earliest = min(earliest, event.EventTime)
				}
			}
			if count == 0 {
				continue
			}
			opened, err := openEventAlert(ctx, queries, group, deploymentID, metric, count, earliest, now)
			if err != nil {
				return nil, err
			}
			if opened {
				response.AlertsOpened++
			}
		}
		for _, event := range pending {
			if err := queries.MarkDeployAnomalyEventProcessed(ctx, db.MarkDeployAnomalyEventProcessedParams{
				ID: event.ID, WorkspaceID: event.WorkspaceID, ProcessedAt: sql.NullInt64{Int64: now, Valid: true},
			}); err != nil {
				return nil, err
			}
			response.EventsProcessed++
		}
		return response, nil
	})
}

func openEventAlert(ctx context.Context, queries *db.Queries, group *hydrav1.DeployAnomalyGroupKey, deploymentID string, metric Metric, count int, earliest, now int64) (bool, error) {
	existing, err := queries.FindOpenAlertEventsByGroup(ctx, db.FindOpenAlertEventsByGroupParams{
		WorkspaceID: group.GetWorkspaceId(), AppID: group.GetAppId(), EnvironmentID: group.GetEnvironmentId(),
	})
	if err != nil {
		return false, err
	}
	for _, alert := range existing {
		if Metric(alert.Metric) == metric {
			return false, queries.TouchAlertEventFromObservedEvent(ctx, db.TouchAlertEventFromObservedEventParams{
				ID: alert.ID, ObservedAt: sql.NullInt64{Int64: now, Valid: true}, ObservedValue: float64(count),
			})
		}
	}
	result := Detect(Input{
		Metric: metric, Current: float64(count), Maximum: 0,
		RequestsInWindow: 0, RecentMedianRequests: 0, RecentActiveBuckets: 0,
		BaselineMean: 0, BaselineStddev: 0, ObservedBaselineBuckets: 0,
		BaselineWindowBuckets: 0, LifetimeStart: 0, PreviousCandidate: false,
	}, DefaultConfig(SensitivityNormal))
	if result.Outcome != OutcomeAnomaly {
		return false, fault.New(fmt.Sprintf("observed %s did not cross its fixed threshold", metric))
	}
	err = queries.InsertAlertEvent(ctx, db.InsertAlertEventParams{
		ID: uid.New(uid.AlertPrefix), WorkspaceID: group.GetWorkspaceId(), ProjectID: group.GetProjectId(),
		AppID: group.GetAppId(), EnvironmentID: group.GetEnvironmentId(),
		DeploymentID: sql.NullString{String: deploymentID, Valid: true},
		Metric:       db.AlertEventsMetric(metric), FiredAt: now, LastSeenAt: now,
		LastObservedEventAt: sql.NullInt64{Int64: now, Valid: true},
		ObservedValue:       result.Observed, BaselineMean: result.BaselineMean,
		BaselineStddev: result.BaselineStddev, ThresholdSigma: result.SigmaK,
		WindowStart: earliest, WindowEnd: now, CreatedAt: now, UpdatedAt: sql.NullInt64{},
	})
	return err == nil, err
}
