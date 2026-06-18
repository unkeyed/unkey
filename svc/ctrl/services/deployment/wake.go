package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// WakeDeployment transitions a stopped deployment back to running. The actual
// Kubernetes apply still happens through the deployment desired-state stream.
func (s *Service) WakeDeployment(ctx context.Context, req *connect.Request[ctrlv1.WakeDeploymentRequest]) (*connect.Response[ctrlv1.WakeDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	deploymentID := req.Msg.GetDeploymentId()
	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}

	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployment: %w", err))
	}

	if deployment.DesiredState != db.DeploymentsDesiredStateStopped {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment is not stopped"))
	}

	logger.Info("waking stopped deployment", "deployment_id", deploymentID)
	_, err = s.deploymentClient(deploymentID).
		WakeDeployment().
		Request(ctx, &hydrav1.WakeDeploymentRequest{
			DeploymentId: deploymentID,
		})
	if err != nil {
		logger.Error("wake deployment workflow failed", "deployment_id", deploymentID, "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("wake deployment workflow failed: %w", err))
	}

	return connect.NewResponse(&ctrlv1.WakeDeploymentResponse{}), nil
}
