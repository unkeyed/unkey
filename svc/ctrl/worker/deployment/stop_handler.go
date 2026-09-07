package deployment

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// StopDeployment sets desired_state=stopped and drops any pending scheduled
// transition. See the package doc for why it does not schedule
func (v *VirtualObject) StopDeployment(ctx restate.ObjectContext, req *hydrav1.StopDeploymentRequest) (*hydrav1.StopDeploymentResponse, error) {
	deploymentID := restate.Key(ctx)
	if id := req.GetDeploymentId(); id != "" && id != deploymentID {
		return nil, restate.TerminalError(fmt.Errorf("deployment_id %q does not match object key %q", id, deploymentID), 400)
	}

	deployment, err := v.loadDeployment(ctx, deploymentID, "stop")
	if err != nil {
		return nil, err
	}

	if err := deploygate.CheckStopTarget(deploygate.StopInput{
		Status:          deployment.Status,
		DesiredState:    deployment.DesiredState,
		EnvironmentKind: deployment.EnvironmentKind,
	}); err != nil {
		return nil, gatefault.Terminal(err)
	}

	if err := v.setDesiredState(ctx, deploymentID, hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STOPPED); err != nil {
		return nil, err
	}

	restate.Clear(ctx, transitionKey)

	if err := audit.InsertDeploymentLifecycle(
		ctx,
		v.auditlogs,
		req.GetActor(),
		req.GetCorrelationId(),
		audit.DeploymentRef{
			ID:            deployment.ID,
			WorkspaceID:   deployment.WorkspaceID,
			ProjectID:     deployment.ProjectID,
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		},
		auditlog.DeploymentStopEvent,
		fmt.Sprintf("Stopped deployment %s", deploymentID),
	); err != nil {
		return nil, fmt.Errorf("insert stop deployment audit log: %w", err)
	}

	return &hydrav1.StopDeploymentResponse{}, nil
}
