package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

// GetDesiredSentinelState returns the target state for a sentinel resource. Krane agents
// use this to determine whether to apply or delete a sentinel. The response contains either
// an ApplySentinel (for running state) or DeleteSentinel (for archived or standby states)
// based on the sentinel's desired_state in the database.
//
// Requires bearer token authentication and the X-Krane-Region header. Returns CodeNotFound
// if the sentinel doesn't exist, or CodeInvalidArgument if the region header is missing.
func (s *Service) GetDesiredSentinelState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelState], error) {

	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sentinel, err := db.Query.FindSentinelByID(ctx, s.db.RO(), req.Msg.GetSentinelId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, err)

		}
		return nil, connect.NewError(connect.CodeInternal, err)

	}

	s.logger.Info("desired sentinel", "state", sentinel.DesiredState)
	switch sentinel.DesiredState {
	case db.SentinelsDesiredStateArchived, db.SentinelsDesiredStateStandby:
		return connect.NewResponse(&ctrlv1.SentinelState{
			State: &ctrlv1.SentinelState_Delete{
				Delete: &ctrlv1.DeleteSentinel{
					K8SName: sentinel.K8sName,
				},
			},
		}), nil
	case db.SentinelsDesiredStateRunning:

		return connect.NewResponse(&ctrlv1.SentinelState{
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
		}), nil
	default:
		s.logger.Error("unhandled sentinel desired state", "desiredState", sentinel.DesiredState)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled sentinel desired state: %s", sentinel.DesiredState))
}
