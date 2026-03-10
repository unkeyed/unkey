package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// cancellableStatuses lists the deployment statuses that can be cancelled.
var cancellableStatuses = map[db.DeploymentsStatus]bool{
	db.DeploymentsStatusPending:    true,
	db.DeploymentsStatusStarting:   true,
	db.DeploymentsStatusBuilding:   true,
	db.DeploymentsStatusDeploying:  true,
	db.DeploymentsStatusNetwork:    true,
	db.DeploymentsStatusFinalizing: true,
}

// CancelDeployment cancels an in-progress deployment by terminating its Restate
// workflow invocation and marking it as cancelled in the database.
func (s *Service) CancelDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CancelDeploymentRequest],
) (*connect.Response[ctrlv1.CancelDeploymentResponse], error) {
	deploymentID := req.Msg.GetDeploymentId()
	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}

	// Look up the deployment
	dep, err := db.Query.FindDeploymentById(ctx, s.db.RO(), deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", deploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find deployment: %w", err))
	}

	// Verify the deployment is in a cancellable state
	if !cancellableStatuses[dep.Status] {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s cannot be cancelled (status: %s)", deploymentID, dep.Status))
	}

	// Cancel the Restate invocation if we have an invocation ID
	if dep.InvocationID.Valid && dep.InvocationID.String != "" {
		if s.restateAdmin == nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("restate admin client is not configured"))
		}

		if killErr := s.restateAdmin.KillInvocation(ctx, dep.InvocationID.String); killErr != nil {
			logger.Warn("failed to kill deployment invocation",
				"deployment_id", deploymentID,
				"invocation_id", dep.InvocationID.String,
				"error", killErr,
			)
			// Continue with status update even if kill fails
		} else {
			logger.Info("killed deployment invocation",
				"deployment_id", deploymentID,
				"invocation_id", dep.InvocationID.String,
			)
		}
	} else {
		logger.Warn("no invocation ID stored for deployment, updating status only",
			"deployment_id", deploymentID,
		)
	}

	// Mark the deployment as cancelled
	err = db.Query.UpdateDeploymentStatus(ctx, s.db.RW(), db.UpdateDeploymentStatusParams{
		ID:        deploymentID,
		Status:    db.DeploymentsStatusCancelled,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment status: %w", err))
	}

	logger.Info("deployment cancelled", "deployment_id", deploymentID)

	return connect.NewResponse(&ctrlv1.CancelDeploymentResponse{}), nil
}
