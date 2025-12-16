package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.GetDesiredDeploymentStateResponse], error) {

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
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	switch deployment.DesiredState {
	case db.DeploymentsDesiredStateArchived, db.DeploymentsDesiredStateStandby:
		return nil, connect.NewError(connect.CodeNotFound, err)
	case db.DeploymentsDesiredStateRunning:
		if deployment.Image.String == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment has no image"))
		}

		return connect.NewResponse(&ctrlv1.GetDesiredDeploymentStateResponse{
			DeploymentId:  deployment.ID,
			Image:         deployment.Image.String,
			Replicas:      deployment.Replicas,
			CpuMillicores: int64(deployment.CpuMillicores),
			MemoryMib:     int64(deployment.MemoryMib),
		}), nil
	default:
		s.logger.Error("unhandled Deployment desired state", "desiredState", deployment.DesiredState)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled Deployment desired state: %s", deployment.DesiredState))
}
