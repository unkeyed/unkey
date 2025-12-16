package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) PullSentinelState(ctx context.Context, req *connect.Request[ctrlv1.PullSentinelStateRequest]) (*connect.Response[ctrlv1.SentinelEvent], error) {

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
			return connect.NewResponse(&ctrlv1.SentinelEvent{
				Event: &ctrlv1.SentinelEvent_Delete{
					Delete: &ctrlv1.DeleteSentinel{
						SentinelId: sentinel.ID,
					},
				},
			}), nil
		}

		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	switch sentinel.DesiredState {
	case db.SentinelsDesiredStateArchived, db.SentinelsDesiredStateStandby:
		return connect.NewResponse(&ctrlv1.SentinelEvent{
			Event: &ctrlv1.SentinelEvent_Delete{
				Delete: &ctrlv1.DeleteSentinel{
					SentinelId: sentinel.ID,
				},
			},
		}), nil
	case db.SentinelsDesiredStateRunning:

		workspace, err := db.Query.FindWorkspaceByID(ctx, s.db.RO(), sentinel.WorkspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return connect.NewResponse(&ctrlv1.SentinelEvent{
			Event: &ctrlv1.SentinelEvent_Apply{
				Apply: &ctrlv1.ApplySentinel{
					Namespace:     workspace.K8sNamespace.String,
					K8SCrdName:    sentinel.K8sCrdName,
					SentinelId:    sentinel.ID,
					WorkspaceId:   sentinel.WorkspaceID,
					ProjectId:     sentinel.ProjectID,
					EnvironmentId: sentinel.EnvironmentID,
					Image:         sentinel.Image,
					Replicas:      uint32(sentinel.DesiredReplicas),
					CpuMillicores: uint32(sentinel.CpuMillicores),
					MemorySizeMib: uint32(sentinel.MemoryMib),
				},
			},
		}), nil
	default:
		s.logger.Error("unhandled sentinel desired state", "desiredState", sentinel.DesiredState)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled sentinel desired state: %s", sentinel.DesiredState))
}
