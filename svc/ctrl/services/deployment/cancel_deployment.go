package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// cancelledByUserMessage is the error message stamped onto any in-flight
// deployment step when a user manually cancels a deployment. The Deploy
// handler's DeploymentStep wrapper may try to end the same step afterwards
// with whatever error the cancellation caused, but EndDeploymentStep only
// updates rows where ended_at IS NULL — so our message wins and the UI
// shows "Cancelled by user" instead of something like "build interrupted".
const cancelledByUserMessage = "Cancelled by user"

// CancelDeployment aborts an in-flight deployment. It stamps any active
// steps with "Cancelled by user", transitions the deployment to the
// cancelled status, then asks Restate to cancel the invocation. The
// compensation stack will try to set status=failed, but
// UpdateDeploymentStatusIfActive protects the cancelled status so the
// compensation is a no-op for the status field while still cleaning up
// partial state (build slots, topologies, routes).
//
// Idempotent:
//   - Deployments already in a terminal status (ready/failed/skipped/stopped)
//     return success without calling Restate.
//   - Deployments without a stored invocation ID are still marked cancelled;
//     the create persists the id just after sending Deploy, so a cancel can
//     land while the column reads NULL. Deploy re-persists the id and checks
//     for a terminal status before building, so the cancelled row stops it.
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

	hasInvocation := deployment.InvocationID.Valid && deployment.InvocationID.String != ""
	if hasInvocation && s.restateAdmin == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("restate admin client is not configured"))
	}

	// Stamp any in-flight deployment steps with "Cancelled by user" BEFORE
	// asking Restate to cancel. This way the step error the UI shows is the
	// reason the user actually triggered, not whatever error the cancellation
	// caused deeper in the workflow (e.g. "build interrupted" or "not enough
	// regions became healthy"). EndDeploymentStep is first-write-wins
	// (WHERE ended_at IS NULL), so the Deploy handler's later attempt to
	// end the same step is a no-op.
	if err := s.db.EndActiveDeploymentStepsWithError(ctx, db.EndActiveDeploymentStepsWithErrorParams{
		DeploymentID: deploymentID,
		EndedAt:      sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		Error:        sql.NullString{Valid: true, String: cancelledByUserMessage},
	}); err != nil {
		// Non-fatal: we still want to cancel the invocation even if this
		// cosmetic update fails. Worst case the UI shows the underlying
		// error instead of "Cancelled by user".
		logger.Warn("failed to mark in-flight steps as cancelled",
			"deployment_id", deploymentID,
			"error", err,
		)
	}

	// Set the status to cancelled before cancelling the invocation. The
	// compensation stack will try UpdateDeploymentStatusIfActive(failed),
	// but cancelled is in the NOT IN list so that update is a no-op.
	if err := s.db.UpdateDeploymentStatusIfActive(ctx, db.UpdateDeploymentStatusIfActiveParams{
		ID:               deploymentID,
		Status:           mysqltype.DeploymentsStatusCancelled,
		UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		TerminalStatuses: mysqltype.TerminalDeploymentStatuses,
	}); err != nil {
		logger.Warn("failed to set deployment status to cancelled",
			"deployment_id", deploymentID,
			"error", err,
		)
	}

	if !hasInvocation {
		logger.Info("cancelled a deployment with no invocation id yet",
			"deployment_id", deploymentID,
		)
		return connect.NewResponse(&ctrlv1.CancelDeploymentResponse{}), nil
	}

	logger.Info("cancelling deployment via restate admin",
		"deployment_id", deploymentID,
		"invocation_id", deployment.InvocationID.String,
	)

	// CancelInvocation treats 404 as success (workflow already finished).
	// Any other error propagates — the caller can retry.
	if err := s.restateAdmin.CancelInvocation(ctx, deployment.InvocationID.String); err != nil {
		logger.Error("failed to cancel restate invocation",
			"deployment_id", deploymentID,
			"invocation_id", deployment.InvocationID.String,
			"error", err,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to cancel: %w", err))
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
