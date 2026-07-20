package deployment

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	pkgdb "github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
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

	if r := deploygate.CheckPromoteTarget(deploygate.PromoteInput{
		Status:              pkgdb.DeploymentsStatus(deployment.Status),
		DesiredState:        pkgdb.DeploymentsDesiredState(deployment.DesiredState),
		EnvironmentSlug:     deployment.EnvironmentSlug,
		CurrentDeploymentID: deployment.CurrentDeploymentID.String,
		DeploymentID:        deployment.ID,
		IsRolledBack:        deployment.IsRolledBack,
	}); r != deploygate.TargetOK {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New(r.Message()))
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
