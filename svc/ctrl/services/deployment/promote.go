package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// Promote reassigns all domains to the target deployment via a Restate workflow.
// The atomic mutation (route reassignment + apps.current_deployment_id update)
// runs inside RoutingService.SwapLiveDeployment, which is per-env serialized.
// The workflow itself is keyed by target deployment_id.
func (s *Service) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	logger.Info("initiating promotion via Restate",
		"target", req.Msg.GetTargetDeploymentId(),
	)

	_, err := s.deploymentClient(req.Msg.GetTargetDeploymentId()).
		Promote().
		Request(ctx, &hydrav1.PromoteRequest{
			TargetDeploymentId: req.Msg.GetTargetDeploymentId(),
		})

	if err != nil {
		logger.Error("promotion workflow failed",
			"target", req.Msg.GetTargetDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("promotion workflow failed: %w", err))
	}

	logger.Info("promotion completed successfully via Restate",
		"target", req.Msg.GetTargetDeploymentId(),
	)

	return connect.NewResponse(&ctrlv1.PromoteResponse{}), nil
}
