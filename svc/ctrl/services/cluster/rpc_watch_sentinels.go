package cluster

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

// WatchSentinels streams sentinel state changes from the control plane to agents.
// Each sentinel controller maintains its own version cursor for resumable streaming.
// The agent applies received state to Kubernetes to converge actual state toward desired state.
//
// This is a long-lived streaming RPC. The server polls the database for new sentinel
// versions and streams them to the client. The client should track the max version seen
// and reconnect with that version to resume from where it left off.
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

	s.logger.Info("krane watching sentinels",
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
			s.logger.Error("failed to fetch sentinel states", "error", err)
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
		s.logger.Error("unhandled sentinel desired state", "desiredState", sentinel.DesiredState)
		return nil
	}
}
