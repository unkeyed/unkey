package deployment

import (
	"fmt"
	"slices"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploycancel"
)

// Written on the open step so the dashboard shows this instead of the Restate
// cancellation error
const cancelledByUserMessage = "Cancelled by user"

func (v *VirtualObject) CancelDeployment(ctx restate.ObjectContext, req *hydrav1.CancelDeploymentRequest) (*hydrav1.CancelDeploymentResponse, error) {
	deploymentID := restate.Key(ctx)
	if id := req.GetDeploymentId(); id != "" && id != deploymentID {
		return nil, restate.ToTerminalError(fmt.Errorf("deployment_id %q does not match object key %q", id, deploymentID), restate.WithErrorCode(400))
	}

	deployment, err := v.loadDeployment(ctx, deploymentID, "cancel")
	if err != nil {
		return nil, err
	}

	if !slices.Contains(mysqltype.ProgressingDeploymentStatuses, deployment.Status) {
		return nil, restate.ToTerminalError(fmt.Errorf("deployment is not in progress, status is %s", deployment.Status), restate.WithErrorCode(412))
	}

	invocationID := deployment.InvocationID.String
	if invocationID != "" && v.restateAdmin == nil {
		return nil, restate.ToTerminalError(fmt.Errorf("restate admin client is not configured"), restate.WithErrorCode(500))
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return deploycancel.Cancel(runCtx, v.db, v.restateAdmin, deploycancel.Params{
			Deployments: []deploycancel.Deployment{{ID: deploymentID, InvocationID: invocationID}},
			Reason:      cancelledByUserMessage,
			Status:      mysqltype.DeploymentsStatusCancelled,
			Audit: &deploycancel.Audit{
				Service:       v.auditlogs,
				Actor:         req.GetActor(),
				CorrelationID: req.GetCorrelationId(),
				WorkspaceID:   deployment.WorkspaceID,
				Meta: map[string]any{
					"projectId":     deployment.ProjectID,
					"appId":         deployment.AppID,
					"environmentId": deployment.EnvironmentID,
				},
			},
		})
	}, restate.WithName("cancel deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("cancel deployment: %w", err)
	}

	return &hydrav1.CancelDeploymentResponse{}, nil
}
