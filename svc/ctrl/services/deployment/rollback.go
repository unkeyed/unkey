package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Rollback switches traffic from the source deployment to a previous target
// deployment via a Restate workflow. This is typically called from the dashboard
// when a deployment needs to be reverted. The workflow runs synchronously
// (blocking until complete) and is keyed by project ID to prevent concurrent
// rollback operations on the same project.
func (s *Service) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error) {
	s.logger.Info("initiating rollback via Restate",
		"source", req.Msg.GetSourceDeploymentId(),
		"target", req.Msg.GetTargetDeploymentId(),
	)

	// Get source deployment to determine project ID for keying
	sourceDeployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetSourceDeploymentId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", req.Msg.GetSourceDeploymentId()))
		}
		s.logger.Error("failed to get deployment",
			"deployment_id", req.Msg.GetSourceDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	// Call the Restate workflow using project ID as the key
	// This ensures only one rollback per project can run at a time
	_, err = s.deploymentClient(sourceDeployment.ProjectID).
		Rollback().
		Request(ctx, &hydrav1.RollbackRequest{
			SourceDeploymentId: req.Msg.GetSourceDeploymentId(),
			TargetDeploymentId: req.Msg.GetTargetDeploymentId(),
		})

	if err != nil {
		s.logger.Error("rollback workflow failed",
			"source", req.Msg.GetSourceDeploymentId(),
			"target", req.Msg.GetTargetDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("rollback workflow failed: %w", err))
	}

	s.logger.Info("rollback completed successfully via Restate",
		"source", req.Msg.GetSourceDeploymentId(),
		"target", req.Msg.GetTargetDeploymentId(),
	)

	return connect.NewResponse(&ctrlv1.RollbackResponse{}), nil
}
