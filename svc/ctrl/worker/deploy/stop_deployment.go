package deploy

import (
	"errors"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
		return w.db.FindDeploymentById(runCtx, deploymentID)
	}, restate.WithName("find deployment for stop"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("deployment not found"), 404)
		}
		return nil, fmt.Errorf("failed to load deployment: %w", err)
	}

	environment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
		return w.db.FindEnvironmentById(runCtx, deployment.EnvironmentID)
	}, restate.WithName("find environment for stop"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("environment not found"), 404)
		}
		return nil, fmt.Errorf("failed to load environment: %w", err)
	}

	if r := deploygate.CheckStoppable(deploygate.StopInput{
		Status:          pkgdb.DeploymentsStatus(deployment.Status),
		DesiredState:    pkgdb.DeploymentsDesiredState(deployment.DesiredState),
		EnvironmentSlug: environment.Slug,
	}); r != deploygate.StopOK {
		return nil, restate.TerminalError(errors.New(r.Message()), 400)
	}

	_, err = hydrav1.NewDeploymentServiceClient(ctx, deploymentID).
		ScheduleDesiredStateChange().
		Request(&hydrav1.ScheduleDesiredStateChangeRequest{
			DelayMillis: 0,
			State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED,
			Overwrite:   true,
		})
	if err != nil {
		return nil, fmt.Errorf("stop deployment workflow failed: %w", err)
	}

	return &hydrav1.StopDeploymentResponse{}, nil
}
