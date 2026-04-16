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

// Rollback switches traffic from the source deployment to a previous target
// deployment via a Restate workflow. The actual atomic mutation
// (route reassignment + apps.current_deployment_id update) is performed
// inside RoutingService.SwapLiveDeployment, which is per-env serialized.
// The workflow itself is keyed by source deployment_id.
func (s *Service) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	logger.Info("initiating rollback via Restate",
		"source", req.Msg.GetSourceDeploymentId(),
		"target", req.Msg.GetTargetDeploymentId(),
	)

	_, err := s.deploymentClient(req.Msg.GetSourceDeploymentId()).
		Rollback().
		Request(ctx, &hydrav1.RollbackRequest{
			SourceDeploymentId: req.Msg.GetSourceDeploymentId(),
			TargetDeploymentId: req.Msg.GetTargetDeploymentId(),
		})

	if err != nil {
		logger.Error("rollback workflow failed",
			"source", req.Msg.GetSourceDeploymentId(),
			"target", req.Msg.GetTargetDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("rollback workflow failed: %w", err))
	}

	logger.Info("rollback completed successfully via Restate",
		"source", req.Msg.GetSourceDeploymentId(),
		"target", req.Msg.GetTargetDeploymentId(),
	)

	return connect.NewResponse(&ctrlv1.RollbackResponse{}), nil
}
