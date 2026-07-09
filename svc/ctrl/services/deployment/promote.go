package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Promote reassigns all domains to the target deployment via a Restate workflow.
// The atomic mutation (route reassignment + apps.current_deployment_id update)
// runs inside RoutingService.SwapLiveDeployment, which is per-env serialized.
// The workflow itself is keyed by target deployment_id.
func (s *Service) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	deploymentID := req.Msg.GetTargetDeploymentId()
	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("target_deployment_id is required"))
	}

	// Validate here so callers get precise connect codes instead of
	// CodeInternal. The workflow re-checks everything except the environment
	// and desired_state gates, which exist only at this layer.
	deployment, err := s.db.FindDeploymentWithEnvironmentAndApp(ctx, deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployment: %w", err))
	}

	// A demoted deployment keeps status ready while it drains toward standby, so
	// status alone would let traffic swap onto one shutting down. Promoting the
	// current live deployment is a no-op unless it is confirming a rollback.
	if err := assert.All(
		assert.Equal(deployment.Status, db.DeploymentsStatusReady, "deployment is not ready"),
		assert.Equal(deployment.DesiredState, db.DeploymentsDesiredStateRunning, "deployment is shutting down"),
		assert.Equal(deployment.EnvironmentSlug, "production", "only production deployments can be promoted"),
		assert.True(deployment.CurrentDeploymentID.Valid, "app has no live deployment"),
		assert.False(deployment.CurrentDeploymentID.String == deployment.ID && !deployment.IsRolledBack, "deployment is already live"),
	); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	logger.Info("initiating promotion via Restate",
		"target", req.Msg.GetTargetDeploymentId(),
	)

	_, err = s.deploymentClient(req.Msg.GetTargetDeploymentId()).
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
