package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Promote reassigns all domains to the target deployment via a Restate workflow.
// This is typically used after a rollback to restore the original deployment, or
// to switch traffic to a new deployment that was previously in a preview state.
// The workflow runs synchronously (blocking until complete) and is keyed by
// project ID to prevent concurrent promotion operations on the same project.
func (s *Service) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error) {
	s.logger.Info("initiating promotion via Restate",
		"target", req.Msg.GetTargetDeploymentId(),
	)

	// Get target deployment to determine project ID for keying
	targetDeployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetTargetDeploymentId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", req.Msg.GetTargetDeploymentId()))
		}
		s.logger.Error("failed to get deployment",
			"deployment_id", req.Msg.GetTargetDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	// Call the Restate workflow using project ID as the key
	// This ensures only one operation per project can run at a time
	_, err = s.deploymentClient(targetDeployment.ProjectID).
		Promote().
		Request(ctx, &hydrav1.PromoteRequest{
			TargetDeploymentId: req.Msg.GetTargetDeploymentId(),
		})

	if err != nil {
		s.logger.Error("promotion workflow failed",
			"target", req.Msg.GetTargetDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("promotion workflow failed: %w", err))
	}

	s.logger.Info("promotion completed successfully via Restate",
		"target", req.Msg.GetTargetDeploymentId(),
	)

	return connect.NewResponse(&ctrlv1.PromoteResponse{}), nil
}
