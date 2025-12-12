package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDesiredGatewayState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredGatewayStateRequest]) (*connect.Response[ctrlv1.GatewayEvent], error) {

	s.logger.Info("get desired gatewaystate", "headers", req.Header())
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	gateway, err := db.Query.FindGatewayByID(ctx, s.db.RO(), req.Msg.GetGatewayId())
	if err != nil {
		if db.IsNotFound(err) {
			return connect.NewResponse(&ctrlv1.GatewayEvent{
				Event: &ctrlv1.GatewayEvent_Delete{
					Delete: &ctrlv1.DeleteGateway{
						GatewayId: gateway.ID,
					},
				},
			}), nil
		}

		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	switch gateway.DesiredState {
	case db.GatewaysDesiredStateArchived, db.GatewaysDesiredStateStandby:
		return connect.NewResponse(&ctrlv1.GatewayEvent{
			Event: &ctrlv1.GatewayEvent_Delete{
				Delete: &ctrlv1.DeleteGateway{
					GatewayId: gateway.ID,
				},
			},
		}), nil
	case db.GatewaysDesiredStateRunning:

		workspace, err := db.Query.FindWorkspaceByID(ctx, s.db.RO(), gateway.WorkspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return connect.NewResponse(&ctrlv1.GatewayEvent{
			Event: &ctrlv1.GatewayEvent_Apply{
				Apply: &ctrlv1.ApplyGateway{
					Namespace:     workspace.K8sNamespace.String,
					K8SCrdName:    gateway.K8sCrdName,
					GatewayId:     gateway.ID,
					WorkspaceId:   gateway.WorkspaceID,
					ProjectId:     gateway.ProjectID,
					EnvironmentId: gateway.EnvironmentID,
					Image:         gateway.Image,
					Replicas:      uint32(gateway.DesiredReplicas),
					CpuMillicores: uint32(gateway.CpuMillicores),
					MemorySizeMib: uint32(gateway.MemoryMib),
				},
			},
		}), nil
	default:
		s.logger.Error("unhandled gateway desired state", "desiredState", gateway.DesiredState)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled gateway desired state: %s", gateway.DesiredState))
}
