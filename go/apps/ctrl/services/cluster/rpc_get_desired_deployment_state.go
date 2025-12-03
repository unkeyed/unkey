package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentEvent], error) {

	s.logger.Info("get desired deploymentstate", "headers", req.Header())
	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	if err := assert.NotEmpty(region, "region is required"); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	deployment, err := db.Query.FindDeploymentTopologyByIDAndRegion(ctx, s.db.RO(), db.FindDeploymentTopologyByIDAndRegionParams{
		DeploymentID: req.Msg.GetDeploymentId(),
		Region:       region,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return connect.NewResponse(&ctrlv1.DeploymentEvent{
				Event: &ctrlv1.DeploymentEvent_Delete{
					Delete: &ctrlv1.DeleteDeployment{
						DeploymentId: deployment.ID,
					},
				},
			}), nil
		}

		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	switch deployment.DesiredState {
	case db.DeploymentsDesiredStateArchived, db.DeploymentsDesiredStateStandby:
		return connect.NewResponse(&ctrlv1.DeploymentEvent{
			Event: &ctrlv1.DeploymentEvent_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					DeploymentId: deployment.ID,
				},
			},
		}), nil
	case db.DeploymentsDesiredStateRunning:
		if deployment.Image.String == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment has no image"))
		}

		return connect.NewResponse(&ctrlv1.DeploymentEvent{
			Event: &ctrlv1.DeploymentEvent_Apply{
				Apply: &ctrlv1.ApplyDeployment{
					DeploymentId:  deployment.ID,
					WorkspaceId:   deployment.WorkspaceID,
					ProjectId:     deployment.ProjectID,
					EnvironmentId: deployment.EnvironmentID,
					Image:         deployment.Image.String,
					Replicas:      uint32(deployment.Replicas),
					CpuMillicores: uint32(deployment.CpuMillicores),
					MemorySizeMib: uint32(deployment.MemoryMib),
				},
			},
		}), nil
	default:
		s.logger.Error("unhandled Deployment desired state", "desiredState", deployment.DesiredState)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled Deployment desired state: %s", deployment.DesiredState))
}
