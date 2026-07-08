package deployment

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
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
	deployment, err := s.db.FindDeploymentById(ctx, deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deployment: %w", err))
	}

	if deployment.Status != db.DeploymentsStatusReady {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment is not ready"))
	}

	// A demoted deployment keeps status ready while it drains toward standby,
	// so status alone would let traffic swap onto a deployment that is
	// shutting down.
	if deployment.DesiredState != db.DeploymentsDesiredStateRunning {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment is shutting down"))
	}

	environment, err := s.db.FindEnvironmentById(ctx, deployment.EnvironmentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("environment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load environment: %w", err))
	}
	// apps.current_deployment_id tracks the production live deployment;
	// promoting outside production would corrupt that pointer.
	if environment.Slug != "production" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("only production deployments can be promoted"))
	}

	app, err := s.db.FindAppById(ctx, deployment.AppID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load app: %w", err))
	}
	if !app.CurrentDeploymentID.Valid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("app has no live deployment"))
	}
	if app.CurrentDeploymentID.String == deployment.ID && !app.IsRolledBack {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deployment is already live"))
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
