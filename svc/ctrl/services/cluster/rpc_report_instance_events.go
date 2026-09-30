package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

const statusObservedAtAttribute = "status_observed_at_unix_nano"

// ReportInstanceEvents buffers lifecycle history and updates application container state.
// Success acknowledges buffering, not durable or exactly-once history delivery.
func (s *Service) ReportInstanceEvents(ctx context.Context, req *connect.Request[ctrlv1.ReportInstanceEventsRequest]) (*connect.Response[ctrlv1.ReportInstanceEventsResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	cluster, err := s.resolveCluster(ctx, req.Msg.GetCluster())
	if err != nil {
		return nil, err
	}
	regionName := cluster.RegionName
	platform := cluster.RegionPlatform

	var firstDenormErr error

	for _, event := range req.Msg.GetEvents() {
		if event == nil {
			continue
		}

		when := eventTime(event)
		observedAtUnixNano := eventObservedAtOrLifecycleTimeUnixNano(event)
		row := schema.InstanceEventV1{
			Time:          when,
			WorkspaceID:   event.GetWorkspaceId(),
			ProjectID:     event.GetProjectId(),
			AppID:         event.GetAppId(),
			EnvironmentID: event.GetEnvironmentId(),
			DeploymentID:  event.GetDeploymentId(),
			PodUID:        event.GetPodUid(),
			PodName:       event.GetPodName(),
			NodeName:      event.GetNodeName(),
			ContainerName: event.GetContainerName(),
			ContainerID:   event.GetContainerId(),
			RestartCount:  event.GetRestartCount(),
			// EventKind, ExitCode, Signal, Reason, Message are set per-case
			// below from the proto's `state` oneof.
			EventKind:        "",
			ExitCode:         0,
			Signal:           0,
			Reason:           "",
			Message:          "",
			Region:           regionName,
			Platform:         platform,
			EventFingerprint: event.GetEventFingerprint(),
			Attributes:       marshalAttributes(event.GetAttributes()),
		}

		// Flatten the proto's `state` oneof into the CH row's flat columns.
		// CH wants a string discriminator + per-row scalar columns; the
		// oneof-on-the-wire shape is for the producer/consumer ergonomics,
		// not the storage layout.
		switch event.GetState().(type) {
		case *ctrlv1.InstanceEvent_Running:
			row.EventKind = "running"
			if event.GetContainerName() != "deployment" {
				break
			}
			err := s.db.ClearInstanceWaiting(ctx, db.ClearInstanceWaitingParams{
				K8sName:            event.GetPodName(),
				RegionID:           cluster.RegionID,
				RestartCount:       int64(event.GetRestartCount()),
				RestartCount_2:     int64(event.GetRestartCount()),
				StatusObservedAt:   observedAtUnixNano,
				StatusObservedAt_2: observedAtUnixNano,
			})
			if err != nil {
				logger.Error("report instance events: clear waiting failed",
					"error", err.Error(),
					"pod_name", event.GetPodName(),
				)
				if firstDenormErr == nil {
					firstDenormErr = err
				}
			}
		case *ctrlv1.InstanceEvent_Terminated:
			t := event.GetTerminated()
			row.EventKind = "terminated"
			row.ExitCode = t.GetExitCode()
			row.Signal = t.GetSignal()
			row.Reason = t.GetReason()
			row.Message = t.GetMessage()

			if t.GetExitCode() == 0 || event.GetContainerName() != "deployment" {
				break
			}

			rc := int64(event.GetRestartCount())
			err := s.db.RecordInstanceExit(ctx, db.RecordInstanceExitParams{
				K8sName:            event.GetPodName(),
				RegionID:           cluster.RegionID,
				StatusObservedAt:   observedAtUnixNano,
				StatusObservedAt_2: observedAtUnixNano,
				RestartCount:       rc,
				RestartCount_2:     rc,
				FinishedAt:         when,
				FinishedAt_2:       when,
				ExitCode:           int64(t.GetExitCode()),
				Signal:             int64(t.GetSignal()),
				Reason:             t.GetReason(),
			})
			if err != nil {
				logger.Error("report instance events: record exit failed",
					"error", err.Error(),
					"pod_name", event.GetPodName(),
					"restart_count", event.GetRestartCount(),
				)
				if firstDenormErr == nil {
					firstDenormErr = err
				}
			}
		case *ctrlv1.InstanceEvent_Waiting:
			w := event.GetWaiting()
			row.EventKind = "waiting"
			row.Reason = w.GetReason()
			row.Message = w.GetMessage()

			if event.GetAttributes()["pod_phase"] == "Failed" || w.GetReason() == "Evicted" {
				observedAt := observedAtUnixNano / int64(time.Millisecond)
				err := s.db.RecordDeploymentPodFailure(ctx, db.RecordDeploymentPodFailureParams{
					DeploymentID: event.GetDeploymentId(),
					WorkspaceID:  event.GetWorkspaceId(),
					PodUid:       event.GetPodUid(),
					PodName:      event.GetPodName(),
					RegionID:     cluster.RegionID,
					Reason:       w.GetReason(),
					Message:      w.GetMessage(),
					ObservedAt:   observedAt,
					ObservedAt_2: observedAt,
				})
				if err != nil && firstDenormErr == nil {
					firstDenormErr = err
				}
				break
			}

			if event.GetContainerName() != "deployment" {
				break
			}
			err := s.db.RecordInstanceWaiting(ctx, db.RecordInstanceWaitingParams{
				K8sName:            event.GetPodName(),
				RegionID:           cluster.RegionID,
				RestartCount:       int64(event.GetRestartCount()),
				RestartCount_2:     int64(event.GetRestartCount()),
				StatusObservedAt:   observedAtUnixNano,
				StatusObservedAt_2: observedAtUnixNano,
				Reason:             w.GetReason(),
				Message:            w.GetMessage(),
			})
			if err != nil {
				logger.Error("report instance events: record waiting failed",
					"error", err.Error(),
					"pod_name", event.GetPodName(),
					"reason", w.GetReason(),
				)
				if firstDenormErr == nil {
					firstDenormErr = err
				}
			}
		default:
			// State unset means the krane producer is on an older proto
			// than ctrl. Skip — there's nothing meaningful to write.
			logger.Warn("report instance events: event has no state set",
				"pod_name", event.GetPodName(),
			)
			continue
		}

		s.instanceEvents.Buffer(row)
	}

	if firstDenormErr != nil {
		// CodeUnavailable signals "transient, retry" to the connect client.
		// Krane's circuit breaker will see this and back off if it persists.
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("instance event denormalization failed: %w", firstDenormErr))
	}

	return connect.NewResponse(&ctrlv1.ReportInstanceEventsResponse{}), nil
}

// eventTime returns the row's CH time, defaulting to "now" if krane sent
// zero so a malformed row doesn't end up with time=0 and partition into the
// epoch bucket.
func eventTime(event *ctrlv1.InstanceEvent) int64 {
	if t := event.GetTime(); t > 0 {
		return t
	}
	return time.Now().UnixMilli()
}

func eventObservedAtOrLifecycleTimeUnixNano(event *ctrlv1.InstanceEvent) int64 {
	if raw := event.GetAttributes()[statusObservedAtAttribute]; raw != "" {
		if observedAtUnixNano, err := strconv.ParseInt(raw, 10, 64); err == nil && observedAtUnixNano > 0 {
			return observedAtUnixNano
		}
	}
	return eventTime(event) * int64(time.Millisecond)
}

// marshalAttributes serializes the proto map into a JSON string for the CH
// JSON column. Returns "{}" for nil/empty input so the column always
// receives valid JSON — an empty Go string would fail the JSON parse on
// insert. Using json.Marshal on a map[string]string is allocation-light
// and emits keys in sorted order, which is friendly to ClickHouse's JSON
// part-merging.
func marshalAttributes(attrs map[string]string) string {
	if len(attrs) == 0 {
		return "{}"
	}
	encoded, err := json.Marshal(attrs)
	if err != nil {
		// json.Marshal of map[string]string can't fail in practice (no
		// channels, funcs, or cycles). Falling back to "{}" keeps the
		// row insertable even if the impossible happens.
		return "{}"
	}
	return string(encoded)
}
