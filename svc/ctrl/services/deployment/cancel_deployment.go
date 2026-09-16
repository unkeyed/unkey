package deployment

import (
	"context"
	"fmt"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploycancel"
)

// Written on the open step so the dashboard shows this instead of the Restate
// cancellation error
const cancelledByUserMessage = "Cancelled by user"

// CancelDeployment aborts an in-flight deployment through deploycancel.Cancel.
// A request without an actor is not audited.
func (s *Service) CancelDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CancelDeploymentRequest],
) (*connect.Response[ctrlv1.CancelDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	deploymentID := req.Msg.GetDeploymentId()
	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}

	deployment, err := s.db.FindDeploymentById(ctx, deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", deploymentID))
		}
		logger.Error("failed to find deployment for cancel",
			"deployment_id", deploymentID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	if deployment.Status.IsTerminal() {
		logger.Info("cancel is a no-op: deployment already terminal",
			"deployment_id", deploymentID,
			"status", deployment.Status,
		)
		return connect.NewResponse(&ctrlv1.CancelDeploymentResponse{}), nil
	}

	invocationID := ""
	if deployment.InvocationID.Valid {
		invocationID = deployment.InvocationID.String
	}
	if invocationID != "" && s.restateAdmin == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("restate admin client is not configured"))
	}

	if err := deploycancel.Cancel(ctx, s.db, s.restateAdmin, deploycancel.Params{
		Deployments: []deploycancel.Deployment{{ID: deploymentID, InvocationID: invocationID}},
		Reason:      cancelledByUserMessage,
		Status:      mysqltype.DeploymentsStatusCancelled,
		Audit: &deploycancel.Audit{
			Service:       s.auditlogs,
			Actor:         req.Msg.GetActor(),
			CorrelationID: "",
			WorkspaceID:   deployment.WorkspaceID,
			// The dashboard's audit feed filters on these keys.
			Meta: map[string]any{
				"projectId":     deployment.ProjectID,
				"appId":         deployment.AppID,
				"environmentId": deployment.EnvironmentID,
			},
		},
	}); err != nil {
		logger.Error("failed to cancel deployment",
			"deployment_id", deploymentID,
			"invocation_id", invocationID,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel: %w", err))
	}

	if invocationID == "" {
		logger.Info("cancelled a deployment with no invocation id yet",
			"deployment_id", deploymentID,
		)
	}

	return connect.NewResponse(&ctrlv1.CancelDeploymentResponse{}), nil
}
