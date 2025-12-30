package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) GetDesiredDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.GetDesiredDeploymentStateRequest]) (*connect.Response[ctrlv1.DeploymentState], error) {

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
		return connect.NewResponse(&ctrlv1.DeploymentState{
			State: &ctrlv1.DeploymentState_Delete{
				Delete: &ctrlv1.DeleteDeployment{
					K8SNamespace: deployment.K8sNamespace.String,
					K8SName:      deployment.K8sName,
				},
			},
		}), nil
	case db.DeploymentsDesiredStateRunning:

		var buildID *string
		if deployment.BuildID.Valid {
			buildID = &deployment.BuildID.String
		}

		return connect.NewResponse(&ctrlv1.DeploymentState{
			State: &ctrlv1.DeploymentState_Apply{
				Apply: &ctrlv1.ApplyDeployment{
					DeploymentId:                  deployment.ID,
					K8SNamespace:                  deployment.K8sNamespace.String,
					K8SName:                       deployment.K8sName,
					WorkspaceId:                   deployment.WorkspaceID,
					ProjectId:                     deployment.ProjectID,
					EnvironmentId:                 deployment.EnvironmentID,
					Replicas:                      deployment.DesiredReplicas,
					Image:                         deployment.Image.String,
					CpuMillicores:                 int64(deployment.CpuMillicores),
					MemoryMib:                     int64(deployment.MemoryMib),
					EncryptedEnvironmentVariables: deployment.EncryptedEnvironmentVariables,
					ReadinessId:                   nil,
					BuildId:                       buildID,
				},
			},
		}), nil
	default:
		s.logger.Error("unhandled Deployment desired state", "desiredState", deployment.DesiredState)
	}

	return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unhandled Deployment desired state: %s", deployment.DesiredState))
}
