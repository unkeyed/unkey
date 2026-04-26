package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
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
//   - Deployments without a stored invocation ID return success (nothing
//     running to cancel; typical for records created before the workflow
//     was kicked off, which shouldn't happen in practice).
//   - Restate returning 404 is treated as success — the workflow already
//     finished in the gap between lookup and cancel.
func (s *Service) CancelDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CancelDeploymentRequest],
) (*connect.Response[ctrlv1.CancelDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}

	deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
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

	if !deployment.InvocationID.Valid || deployment.InvocationID.String == "" {
		logger.Info("cancel is a no-op: deployment has no invocation id",
			"deployment_id", deploymentID,
		)
		return connect.NewResponse(&ctrlv1.CancelDeploymentResponse{}), nil
	}

	if s.restateAdmin == nil {
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
	if err := db.Query.EndActiveDeploymentStepsWithError(ctx, s.db.RW(), db.EndActiveDeploymentStepsWithErrorParams{
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

	logger.Info("cancelling deployment via restate admin",
		"deployment_id", deploymentID,
		"invocation_id", deployment.InvocationID.String,
	)

	// Set the status to cancelled before cancelling the invocation. The
	// compensation stack will try UpdateDeploymentStatusIfActive(failed),
	// but cancelled is in the NOT IN list so that update is a no-op.
	if err := db.Query.UpdateDeploymentStatusIfActive(ctx, s.db.RW(), db.UpdateDeploymentStatusIfActiveParams{
		ID:        deploymentID,
		Status:    db.DeploymentsStatusCancelled,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); err != nil {
		logger.Warn("failed to set deployment status to cancelled",
			"deployment_id", deploymentID,
			"error", err,
		)
	}

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

	// Belt-and-suspenders: the Restate cancel above will (on the happy path)
	// trigger the Deploy workflow's compensation stack, which releases the
	// build slot. But if that compensation doesn't run — Restate
	// hard-terminates the invocation without replay, or the Send from the
	// compensation is dropped — the slot leaks and subsequent deployments
	// pile up on the wait list forever. Fire Release directly here as an
	// independent signal. Release is idempotent, so when compensation does
	// run, the second call is a no-op.
	if s.restate != nil {
		if _, err := hydrav1.NewBuildSlotServiceIngressClient(s.restate, deployment.WorkspaceID).
			Release().
			Request(ctx, &hydrav1.ReleaseSlotRequest{DeploymentId: deploymentID}); err != nil {
			// Not fatal to the cancel operation — the workflow compensation
			// will likely handle it, and the AcquireOrWait self-heal is the
			// final safety net.
			logger.Warn("belt-and-suspenders Release after cancel failed",
				"deployment_id", deploymentID,
				"workspace_id", deployment.WorkspaceID,
				"error", err,
			)
		}
	}

	return connect.NewResponse(&ctrlv1.CancelDeploymentResponse{}), nil
}

// isTerminalDeploymentStatus reports whether a deployment status is one
// from which no further state transitions will happen. Cancelling a
// terminal deployment is a no-op.
func isTerminalDeploymentStatus(status db.DeploymentsStatus) bool {
	switch status {
	case db.DeploymentsStatusReady,
		db.DeploymentsStatusFailed,
		db.DeploymentsStatusSkipped,
		db.DeploymentsStatusStopped,
		db.DeploymentsStatusSuperseded,
		db.DeploymentsStatusCancelled:
		return true
	case db.DeploymentsStatusPending,
		db.DeploymentsStatusStarting,
		db.DeploymentsStatusBuilding,
		db.DeploymentsStatusDeploying,
		db.DeploymentsStatusNetwork,
		db.DeploymentsStatusFinalizing,
		db.DeploymentsStatusAwaitingApproval:
		return false
	default:
		return false
	}
}
