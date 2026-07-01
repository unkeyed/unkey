package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	dbtype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// ReportInstanceEvents persists container lifecycle events captured by a
// krane agent. The handler does two writes per event:
//
//  1. Buffered insert into ClickHouse instance_events_raw_v1 — the canonical
//     event log queried by the dashboard's events panel and the logs viewer
//     enrichment path.
//  2. UPDATE on the matching instances row in MySQL with the latest exit
//     metadata. The dashboard header reads these denormalized fields so a
//     deployment summary card never round-trips to ClickHouse.
//
// Events from a single pod-watch tick are batched into one RPC. The CH
// insert is buffered through the service-wide batch processor, so a single
// RPC almost never blocks on CH; MySQL writes are immediate but cheap (one
// row per event keyed by k8s_name + region_id).
//
// Krane's in-memory LRU prevents most duplicate sends, and ClickHouse's
// insert-deduplication window catches retries inside that table; the MySQL
// guard on (restart_count, last_exit_finished_at) prevents an out-of-order
// RPC from clobbering a newer exit. The handler therefore tolerates
// best-effort delivery from krane.
//
// CH inserts always happen — the canonical event log must capture every
// reported event regardless of denorm outcome. MySQL denorm errors mark
// the *whole* RPC as failed (CodeUnavailable) so krane re-emits on its
// next watch tick. The CH-side insert dedupe window absorbs the resulting
// duplicate inserts, and the MySQL guards make the writes idempotent.
//
// Returns CodeUnauthenticated if bearer token is invalid; CodeInvalidArgument
// if region/platform headers are missing or the region is unknown;
// CodeUnavailable if any MySQL denorm write failed during the batch.
func (s *Service) ReportInstanceEvents(ctx context.Context, req *connect.Request[ctrlv1.ReportInstanceEventsRequest]) (*connect.Response[ctrlv1.ReportInstanceEventsResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	region, err := s.resolveRegion(ctx, req.Msg.GetRegion())
	if err != nil {
		return nil, err
	}
	regionName := region.Name
	platform := region.Platform

	// firstDenormErr holds the first MySQL denorm error encountered. The
	// loop keeps going past it so every reported event still lands in CH;
	// after the loop we surface CodeUnavailable to the caller so krane
	// retries. Subsequent retries hit the CH dedup window and the MySQL
	// guards, both of which are idempotent.
	var firstDenormErr error

	for _, event := range req.Msg.GetEvents() {
		if event == nil {
			continue
		}

		when := eventTime(event)
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
			// Running carries no exit metadata — the row is identity +
			// time + attributes, written for the dashboard's "container
			// booted" timeline divider. No MySQL denormalization: the live
			// instance row is already tracked via reportDeploymentStatus.
			row.EventKind = "running"
		case *ctrlv1.InstanceEvent_Terminated:
			t := event.GetTerminated()
			row.EventKind = "terminated"
			row.ExitCode = t.GetExitCode()
			row.Signal = t.GetSignal()
			row.Reason = t.GetReason()
			row.Message = t.GetMessage()

			// Only failures denormalize onto the instances row. Clean
			// exits (init-container migrations, cron pods, graceful
			// shutdowns with exit 0) still land in CH for the timeline
			// divider, but must not flow into container_status — the
			// dashboard renders that as "last failure" and would show a
			// red "Completed" badge for a healthy init completion.
			if t.GetExitCode() == 0 {
				break
			}

			// Build the post-update ContainerStatus in one place. The new
			// value fully replaces what's on the row (including clearing
			// $.waiting, since a fresh exit ends any prior crashloop
			// window). The stale-event guard in WHERE rejects this
			// payload if a newer event has already won.
			rc := int64(event.GetRestartCount())
			//nolint:gosec // krane reports int32 (kubelet's type); the
			// column stores uint32 since restart counts can never be
			// negative. Conversion is safe for any value krane sends.
			newStatus := dbtype.ContainerStatus{
				RestartCount: uint32(event.GetRestartCount()),
				LastTerminationState: &dbtype.TerminatedState{
					ExitCode:   t.GetExitCode(),
					Signal:     t.GetSignal(),
					Reason:     t.GetReason(),
					FinishedAt: when,
				},
				Waiting: nil,
			}
			err := s.db.RecordInstanceExit(ctx, db.RecordInstanceExitParams{
				ContainerStatus: newStatus,
				K8sName:         event.GetPodName(),
				RegionID:        region.ID,
				// restart_count appears twice in the WHERE clause — sqlc
				// emits a separate field per positional placeholder. Same
				// value passed to both.
				RestartCount:   rc,
				RestartCount_2: rc,
				FinishedAt:     when,
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

			// Only CrashLoopBackOff drives the MySQL "instance is in
			// crashloop" denormalization. Other waiting reasons (image
			// pull errors, ContainerCreating, …) flow into CH as raw rows
			// for the timeline but don't change the live instance summary
			// the dashboard header reads.
			if w.GetReason() == "CrashLoopBackOff" {
				err := s.db.RecordInstanceCrashLoopBackOff(ctx, db.RecordInstanceCrashLoopBackOffParams{
					K8sName:      event.GetPodName(),
					RegionID:     region.ID,
					RestartCount: int64(event.GetRestartCount()),
				})
				if err != nil {
					logger.Error("report instance events: record crashloop failed",
						"error", err.Error(),
						"pod_name", event.GetPodName(),
					)
					if firstDenormErr == nil {
						firstDenormErr = err
					}
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
