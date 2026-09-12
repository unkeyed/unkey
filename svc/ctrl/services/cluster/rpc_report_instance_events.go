package cluster

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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

const deployAnomalyInboxWriteTimeout = 5 * time.Second

type deployAnomalyEventInserter interface {
	InsertDeployAnomalyEvent(context.Context, db.InsertDeployAnomalyEventParams) error
}

// ReportInstanceEvents persists container lifecycle events reported by krane.
// It preserves the ClickHouse event log and instance-row denormalization. For
// allowlisted workspaces it also durably records OOMKilled and CrashLoopBackOff
// identities before returning success. Duplicate identities do not change an
// existing inbox row.
//
// It returns CodeInvalidArgument for invalid qualifying identities and
// CodeUnavailable when a required MySQL write fails.
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

	// firstDenormErr holds the first MySQL denorm error encountered. The
	// loop keeps going past it so every reported event still lands in CH;
	// after the loop we surface CodeUnavailable to the caller so krane
	// retries. Subsequent retries hit the CH dedup window and the MySQL
	// guards, both of which are idempotent.
	var firstDenormErr error
	var firstInboxErr error
	inboxCtx, cancelInboxWrites := context.WithTimeout(ctx, deployAnomalyInboxWriteTimeout)
	defer cancelInboxWrites()

	for _, event := range req.Msg.GetEvents() {
		if event == nil {
			continue
		}

		when := eventTime(event)
		inboxEvent, qualifies, inboxErr := s.deployAnomalyEvent(event, regionName, when)
		if inboxErr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, inboxErr)
		}
		if qualifies {
			if err := s.deployAnomalyEvents.InsertDeployAnomalyEvent(inboxCtx, inboxEvent); err != nil {
				logger.Error("report instance events: insert anomaly event failed",
					"error", err.Error(),
					"workspace_id", event.GetWorkspaceId(),
					"deployment_id", event.GetDeploymentId(),
				)
				if firstInboxErr == nil {
					firstInboxErr = err
				}
			}
		}

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
				RegionID:        cluster.RegionID,
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
					RegionID:     cluster.RegionID,
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

	if firstInboxErr != nil {
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("deploy anomaly inbox write failed: %w", firstInboxErr))
	}

	if firstDenormErr != nil {
		// CodeUnavailable signals "transient, retry" to the connect client.
		// Krane's circuit breaker will see this and back off if it persists.
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("instance event denormalization failed: %w", firstDenormErr))
	}

	return connect.NewResponse(&ctrlv1.ReportInstanceEventsResponse{}), nil
}

func (s *Service) deployAnomalyEvent(event *ctrlv1.InstanceEvent, region string, eventTime int64) (db.InsertDeployAnomalyEventParams, bool, error) {
	var empty db.InsertDeployAnomalyEventParams
	if _, enabled := s.deployAnomalyFastWorkspaces[event.GetWorkspaceId()]; !enabled {
		return empty, false, nil
	}

	var metric db.DeployAnomalyEventsMetric
	var eventKind string
	switch event.GetState().(type) {
	case *ctrlv1.InstanceEvent_Terminated:
		if event.GetTerminated().GetReason() != "OOMKilled" {
			return empty, false, nil
		}
		eventKind = "terminated"
		metric = db.DeployAnomalyEventsMetricOomKilled
	case *ctrlv1.InstanceEvent_Waiting:
		if event.GetWaiting().GetReason() != "CrashLoopBackOff" {
			return empty, false, nil
		}
		eventKind = "waiting"
		metric = db.DeployAnomalyEventsMetricCrashLoop
	default:
		return empty, false, nil
	}

	identity := []struct {
		name  string
		value string
	}{
		{name: "workspace_id", value: event.GetWorkspaceId()},
		{name: "project_id", value: event.GetProjectId()},
		{name: "app_id", value: event.GetAppId()},
		{name: "environment_id", value: event.GetEnvironmentId()},
		{name: "deployment_id", value: event.GetDeploymentId()},
		{name: "region", value: region},
		{name: "pod_uid", value: event.GetPodUid()},
		{name: "container_name", value: event.GetContainerName()},
	}
	for _, field := range identity {
		if field.value == "" {
			return empty, false, fmt.Errorf("deploy anomaly event %s is required", field.name)
		}
	}
	if event.GetRestartCount() < 0 {
		return empty, false, fmt.Errorf("deploy anomaly event restart_count must be nonnegative")
	}

	return db.InsertDeployAnomalyEventParams{
		ID: deployAnomalyEventID(
			event.GetWorkspaceId(),
			event.GetDeploymentId(),
			region,
			event.GetPodUid(),
			event.GetContainerName(),
			event.GetRestartCount(),
			eventKind,
		),
		WorkspaceID:   event.GetWorkspaceId(),
		ProjectID:     event.GetProjectId(),
		AppID:         event.GetAppId(),
		EnvironmentID: event.GetEnvironmentId(),
		DeploymentID:  event.GetDeploymentId(),
		Metric:        metric,
		EventTime:     eventTime,
		ReceivedAt:    time.Now().UnixMilli(),
	}, true, nil
}

func deployAnomalyEventID(workspaceID, deploymentID, region, podUID, containerName string, restartCount int32, eventKind string) string {
	encoded := make([]byte, 0, 256)
	for _, value := range []string{workspaceID, deploymentID, region, podUID, containerName} {
		encoded = binary.BigEndian.AppendUint32(encoded, uint32(len(value)))
		encoded = append(encoded, value...)
	}
	encoded = binary.BigEndian.AppendUint32(encoded, uint32(restartCount))
	encoded = binary.BigEndian.AppendUint32(encoded, uint32(len(eventKind)))
	encoded = append(encoded, eventKind...)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
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
