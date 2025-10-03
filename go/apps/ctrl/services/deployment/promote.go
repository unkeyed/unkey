package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// Promote reassigns all domains to a deployment and removes the rolled back state via Restate workflow
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
	_, err = restateingress.Object[*hydrav1.PromoteRequest, *hydrav1.PromoteResponse](
		s.restate,
		"hydra.v1.DeploymentService",
		targetDeployment.ProjectID,
		"Promote",
	).Request(ctx, &hydrav1.PromoteRequest{
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
