package deployanomaly

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const (
	eventPageSize           = 100
	eventPagesPerWorkspace  = 10
	eventDispatchConcurrent = 4
)

type EventsConfig struct {
	DB         db.Database
	Workspaces []string
	Heartbeat  healthcheck.Heartbeat
}

type EventsHandler struct {
	db         db.Database
	workspaces []string
	heartbeat  healthcheck.Heartbeat
}

type eventCursor struct {
	LastScannedPK  uint64 `json:"after"`
	LastEligiblePK uint64 `json:"through"`
}

func NewEventsHandler(cfg EventsConfig) (*EventsHandler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	workspaces := slices.Clone(cfg.Workspaces)
	slices.Sort(workspaces)
	return &EventsHandler{db: cfg.DB, workspaces: slices.Compact(workspaces), heartbeat: cfg.Heartbeat}, nil
}

func EventsRetryPolicy() restate.HandlerOption {
	return restate.WithInvocationRetryPolicy(
		restate.WithInitialInterval(100*time.Millisecond),
		restate.WithExponentiationFactor(2.0),
		restate.WithMaxInterval(5*time.Second),
		restate.WithMaxAttempts(5),
		restate.KillOnMaxAttempts(),
	)
}

func (h *EventsHandler) Handle(ctx restate.ObjectContext, _ *hydrav1.RunDeployAnomalyEventsRequest) (*hydrav1.RunDeployAnomalyEventsResponse, error) {
	if restate.Key(ctx) != "deploy-anomaly-events" {
		return nil, restate.TerminalError(fault.New("invalid deploy anomaly events key"))
	}
	response := &hydrav1.RunDeployAnomalyEventsResponse{}
	var failures error
	for _, workspaceID := range h.workspaces {
		cursorKey := "events_cursor:" + workspaceID
		cursor, err := restate.Get[eventCursor](ctx, cursorKey)
		if err != nil {
			return nil, err
		}
		if cursor.LastEligiblePK == 0 {
			through, err := restate.Run(ctx, func(rc restate.RunContext) (uint64, error) {
				queryCtx, cancel := context.WithTimeout(rc, 10*time.Second)
				defer cancel()
				maximum, err := h.db.FindPendingDeployAnomalyEventMaxPk(queryCtx, workspaceID)
				if err != nil {
					return 0, err
				}
				if maximum < 0 {
					return 0, fault.New("anomaly event pk exceeds supported cursor range")
				}
				return uint64(maximum), nil
			}, restate.WithName("bound anomaly event sweep"))
			if err != nil {
				failures = errors.Join(failures, err)
				continue
			}
			cursor.LastEligiblePK = through
			restate.Set(ctx, cursorKey, cursor)
		}
		for page := range eventPagesPerWorkspace {
			rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.DeployAnomalyEvent, error) {
				queryCtx, cancel := context.WithTimeout(rc, 10*time.Second)
				defer cancel()
				return h.db.ListPendingDeployAnomalyEvents(queryCtx, db.ListPendingDeployAnomalyEventsParams{
					WorkspaceID: workspaceID, AfterPk: cursor.LastScannedPK, ThroughPk: cursor.LastEligiblePK, Limit: eventPageSize,
				})
			}, restate.WithName("list pending anomaly events"))
			if err != nil {
				failures = errors.Join(failures, err)
				break
			}
			result, dispatchErr := dispatchEventPage(ctx, rows)
			failures = errors.Join(failures, dispatchErr)
			response.EventsProcessed += result.GetEventsProcessed()
			response.AlertsOpened += result.GetAlertsOpened()
			if len(rows) < eventPageSize {
				restate.Clear(ctx, cursorKey)
				break
			}
			cursor.LastScannedPK = rows[len(rows)-1].Pk
			restate.Set(ctx, cursorKey, cursor)
			if page == eventPagesPerWorkspace-1 {
				logger.Warn("fast anomaly event scan reached per-tick capacity",
					"workspace_id", workspaceID, "after_pk", cursor.LastScannedPK, "through_pk", cursor.LastEligiblePK)
			}
		}
	}
	if failures != nil {
		return nil, fault.Wrap(failures, fault.Internal("dispatch pending anomaly events"))
	}
	if len(h.workspaces) > 0 {
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error { return h.heartbeat.Ping(rc) }, restate.WithName("send fast anomaly heartbeat")); err != nil {
			return nil, err
		}
	}
	logger.Info("fast anomaly event scan complete", "events_processed", response.EventsProcessed, "alerts_opened", response.AlertsOpened)
	return response, nil
}

func dispatchEventPage(ctx restate.ObjectContext, rows []db.DeployAnomalyEvent) (*hydrav1.RunDeployAnomalyEventsResponse, error) {
	groups := make(map[string]*hydrav1.OpenObservedDeployAnomalyEventsRequest)
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		key := GroupKey(row.WorkspaceID, row.AppID, row.EnvironmentID)
		if _, ok := groups[key]; !ok {
			keys = append(keys, key)
			groups[key] = &hydrav1.OpenObservedDeployAnomalyEventsRequest{
				Group: &hydrav1.DeployAnomalyGroupKey{
					WorkspaceId: row.WorkspaceID, ProjectId: row.ProjectID, AppId: row.AppID, EnvironmentId: row.EnvironmentID,
				},
			}
		}
		groups[key].EventIds = append(groups[key].EventIds, row.ID)
	}
	response := &hydrav1.RunDeployAnomalyEventsResponse{}
	var failures error
	for start := 0; start < len(keys); start += eventDispatchConcurrent {
		batch := keys[start:min(start+eventDispatchConcurrent, len(keys))]
		futures := make([]restate.ResponseFuture[*hydrav1.OpenObservedDeployAnomalyEventsResponse], len(batch))
		for i, key := range batch {
			futures[i] = hydrav1.NewDeployAnomalyServiceClient(ctx, key).OpenObservedEvents().RequestFuture(groups[key])
		}
		for i, future := range futures {
			result, err := future.Response()
			if err != nil {
				failures = errors.Join(failures, fmt.Errorf("group %s: %w", batch[i], err))
				continue
			}
			response.EventsProcessed += result.GetEventsProcessed()
			response.AlertsOpened += result.GetAlertsOpened()
		}
	}
	return response, failures
}
