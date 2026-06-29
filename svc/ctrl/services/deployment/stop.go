package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// StopDeployment transitions a running deployment to stopped. The actual
// Kubernetes apply still happens through the deployment desired-state stream.
func (s *Service) StopDeployment(ctx context.Context, req *connect.Request[ctrlv1.StopDeploymentRequest]) (*connect.Response[ctrlv1.StopDeploymentResponse], error) {
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

	err = assert.All(
		assert.Equal(deployment.Status, db.DeploymentsStatusReady, "deployment is not running"),
		assert.Equal(deployment.DesiredState, db.DeploymentsDesiredStateRunning, "deployment is not running"),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	environment, err := db.Query.FindEnvironmentById(ctx, s.db.RO(), deployment.EnvironmentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("environment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load environment: %w", err))
	}

	err = assert.All(
		assert.NotEqual(environment.Slug, "production", "production deployments cannot be stopped"),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	logger.Info("stopping deployment", "deployment_id", deploymentID)
	_, err = s.deploymentClient(deploymentID).
		StopDeployment().
		Request(ctx, &hydrav1.StopDeploymentRequest{
			DeploymentId: deploymentID,
		})
	if err != nil {
		logger.Error("stop deployment workflow failed", "deployment_id", deploymentID, "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("stop deployment workflow failed: %w", err))
	}

	return connect.NewResponse(&ctrlv1.StopDeploymentResponse{}), nil
}
