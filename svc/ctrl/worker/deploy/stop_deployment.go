package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// StopDeployment is the public Restate entrypoint for putting a running
// deployment into stopped. The actual desired-state mutation remains serialized
// in the private DeploymentService VO keyed by deployment_id.
func (w *Workflow) StopDeployment(ctx restate.ObjectContext, req *hydrav1.StopDeploymentRequest) (*hydrav1.StopDeploymentResponse, error) {
	deploymentID := req.GetDeploymentId()
	if deploymentID == "" {
		return nil, restate.TerminalError(fmt.Errorf("deployment_id is required"), 400)
	}

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(runCtx, w.db.RO(), deploymentID)
	}, restate.WithName("find deployment for stop"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("deployment not found"), 404)
		}
		return nil, fmt.Errorf("failed to load deployment: %w", err)
	}

	if deployment.Status != db.DeploymentsStatusReady {
		return nil, restate.TerminalError(fmt.Errorf("deployment is not ready"), 400)
	}

	environment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return db.Query.FindEnvironmentById(runCtx, w.db.RO(), deployment.EnvironmentID)
	}, restate.WithName("find environment for stop"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("environment not found"), 404)
		}
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}
	if environment.Slug == "production" {
		return nil, restate.TerminalError(fmt.Errorf("production deployments cannot be stopped"), 400)
	}

	if deployment.DesiredState != db.DeploymentsDesiredStateRunning {
		return nil, restate.TerminalError(fmt.Errorf("deployment is not running"), 400)
	}

	_, err = hydrav1.NewDeploymentServiceClient(ctx, deploymentID).
		ScheduleDesiredStateChange().
		Request(&hydrav1.ScheduleDesiredStateChangeRequest{
			DelayMillis: 0,
			State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
		})
	if err != nil {
		return nil, fmt.Errorf("stop deployment workflow failed: %w", err)
	}

	return &hydrav1.StopDeploymentResponse{}, nil
}
