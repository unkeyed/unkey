package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// StopDeployment is the public Restate entrypoint for putting a running
// deployment into stopped. It runs on the deployment's own virtual object and
// applies the desired-state mutation itself, so a success response means the
// state actually changed.
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

	if err := deploygate.CheckStopTarget(deploygate.StopInput{
		Status:          deployment.Status,
		DesiredState:    deployment.DesiredState,
		EnvironmentKind: environment.Kind,
	}); err != nil {
		return nil, gatefault.Terminal(err)
	}

	// Same-key, so the state change applies before returning; see applyDesiredStateNow.
	if err := w.applyDesiredStateNow(ctx, deploymentID, hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED); err != nil {
		return nil, fmt.Errorf("stop deployment workflow failed: %w", err)
	}

	if err := w.insertLifecycleAudit(
		ctx,
		req.GetActor(),
		req.GetCorrelationId(),
		deployment,
		auditlog.DeploymentStopEvent,
		fmt.Sprintf("Stopped deployment %s", deploymentID),
	); err != nil {
		return nil, fmt.Errorf("insert stop deployment audit log: %w", err)
	}

	return &hydrav1.StopDeploymentResponse{}, nil
}
