package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// WatchSentinels streams sentinel state changes from the control plane to agents.
// This is the primary mechanism for agents to receive desired state updates for their region.
// Agents apply received state to Kubernetes to converge actual state toward desired state.
//
// The stream uses version-based cursors for resumability. The client provides version_last_seen
// in the request, and the server streams all sentinels with versions greater than that cursor.
// Clients should track the maximum version received and use it to reconnect without replaying
// history. When no new versions are available, the server polls the database every second.
//
// Each poll fetches up to 100 sentinel rows ordered by version. The desired_state field
// determines whether to send an ApplySentinel (for running state) or DeleteSentinel (for
// archived or standby states). Rows with unhandled states are logged and skipped.
//
// Returns when the context is cancelled, or on database or stream errors.
func (s *Service) WatchSentinels(
	ctx context.Context,
	req *connect.Request[ctrlv1.WatchSentinelsRequest],
	stream *connect.ServerStream[ctrlv1.SentinelState],
) error {
	if err := s.authenticate(req); err != nil {
		return err
	}

	region := req.Msg.GetRegion()
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	versionCursor := req.Msg.GetVersionLastSeen()

	logger.Info("krane watching sentinels",
		"region", region,
		"version", versionCursor,
	)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		states, err := s.fetchSentinelStates(ctx, region, versionCursor)
		if err != nil {
			logger.Error("failed to fetch sentinel states", "error", err)
			return connect.NewError(connect.CodeInternal, err)
		}

		for _, state := range states {
			if err := stream.Send(state); err != nil {
				return err
			}
			if state.GetVersion() > versionCursor {
				versionCursor = state.GetVersion()
			}
		}

		if len(states) == 0 {
			time.Sleep(time.Second)
		}
	}
}

// fetchSentinelStates queries the database for sentinels in the given region with versions
// greater than afterVersion, returning up to 100 results. Rows with unhandled desired_state
// values are skipped rather than failing the entire batch.
func (s *Service) fetchSentinelStates(ctx context.Context, region string, afterVersion uint64) ([]*ctrlv1.SentinelState, error) {
	rows, err := db.Query.ListSentinelsByRegion(ctx, s.db.RO(), db.ListSentinelsByRegionParams{
		Region:       region,
		Afterversion: afterVersion,
		Limit:        100,
	})
	if err != nil {
		return nil, err
	}

	states := make([]*ctrlv1.SentinelState, 0, len(rows))
	for _, row := range rows {
		state := s.sentinelRowToState(row)
		if state != nil {
			states = append(states, state)
		}
	}

	return states, nil
}

// sentinelRowToState converts a database sentinel row to a proto SentinelState message.
// Returns a DeleteSentinel for archived or standby states and an ApplySentinel for running
// state. Returns nil for unhandled states, which the caller should skip.
func (s *Service) sentinelRowToState(sentinel db.Sentinel) *ctrlv1.SentinelState {
	switch sentinel.DesiredState {
	case db.SentinelsDesiredStateArchived, db.SentinelsDesiredStateStandby:
		return &ctrlv1.SentinelState{
			Version: sentinel.Version,
			State: &ctrlv1.SentinelState_Delete{
				Delete: &ctrlv1.DeleteSentinel{
					K8SName: sentinel.K8sName,
				},
			},
		}
	case db.SentinelsDesiredStateRunning:
		return &ctrlv1.SentinelState{
			Version: sentinel.Version,
			State: &ctrlv1.SentinelState_Apply{
				Apply: &ctrlv1.ApplySentinel{
					SentinelId:    sentinel.ID,
					K8SName:       sentinel.K8sName,
					WorkspaceId:   sentinel.WorkspaceID,
					ProjectId:     sentinel.ProjectID,
					EnvironmentId: sentinel.EnvironmentID,
					Replicas:      sentinel.DesiredReplicas,
					Image:         sentinel.Image,
					CpuMillicores: int64(sentinel.CpuMillicores),
					MemoryMib:     int64(sentinel.MemoryMib),
				},
			},
		}
	default:
		logger.Error("unhandled sentinel desired state", "desiredState", sentinel.DesiredState)
		return nil
	}
}
