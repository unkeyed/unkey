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

// cancelledByUserMessage is the error message stamped onto any in-flight
// deployment step when a user manually cancels a deployment. The Deploy
// handler's DeploymentStep wrapper may try to end the same step afterwards
// with whatever error the cancellation caused, but EndDeploymentStep only
// updates rows where ended_at IS NULL — so our message wins and the UI
// shows "Cancelled by user" instead of something like "build interrupted".
const cancelledByUserMessage = "Cancelled by user"

// CancelDeployment aborts an in-flight deployment through
// [deploycancel.Cancel]: active steps are stamped with "Cancelled by user", the
// row transitions to cancelled, Restate is asked to cancel the invocation, and
// the cancel is audited with the actor the request carries. A request without
// an actor is not audited.
//
// Idempotent:
//   - Deployments already in a terminal status (ready/failed/skipped/stopped)
//     return success without calling Restate.
//   - Deployments without a stored invocation ID are still marked cancelled:
//     the id is persisted just after Deploy is sent, so a cancel can land
//     while the column reads NULL, and Deploy checks for a terminal status
//     before it builds.
//   - Restate returning 404 is treated as success — the workflow already
//     finished in the gap between lookup and cancel.
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

	if isTerminalDeploymentStatus(deployment.Status) {
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

	// This check stays out of [deploycancel.Cancel]: a nil *Client wrapped in a
	// non-nil interface value passes its admin != nil guard and then panics.
	var canceler deploycancel.InvocationCanceler
	if s.restateAdmin != nil {
		canceler = s.restateAdmin
	}

	if err := deploycancel.Cancel(ctx, s.db, canceler, deploycancel.Params{
		Targets: []deploycancel.Target{{ID: deploymentID, InvocationID: invocationID}},
		Reason:  cancelledByUserMessage,
		Status:  mysqltype.DeploymentsStatusCancelled,
		Audit: &deploycancel.Audit{
			Service:       s.auditlogs,
			Actor:         req.Msg.GetActor(),
			CorrelationID: "",
			WorkspaceID:   deployment.WorkspaceID,
			Meta:          deploymentAuditMeta(deployment.ProjectID, deployment.AppID, deployment.EnvironmentID),
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

// isTerminalDeploymentStatus reports whether a deployment status is one
// from which no further state transitions will happen. Cancelling a
// terminal deployment is a no-op.
func isTerminalDeploymentStatus(status mysqltype.DeploymentsStatus) bool {
	switch status {
	case mysqltype.DeploymentsStatusReady,
		mysqltype.DeploymentsStatusFailed,
		mysqltype.DeploymentsStatusSkipped,
		mysqltype.DeploymentsStatusStopped,
		mysqltype.DeploymentsStatusSuperseded,
		mysqltype.DeploymentsStatusCancelled:
		return true
	case mysqltype.DeploymentsStatusPending,
		mysqltype.DeploymentsStatusStarting,
		mysqltype.DeploymentsStatusBuilding,
		mysqltype.DeploymentsStatusDeploying,
		mysqltype.DeploymentsStatusNetwork,
		mysqltype.DeploymentsStatusFinalizing,
		mysqltype.DeploymentsStatusAwaitingApproval:
		return false
	default:
		return false
	}
}
