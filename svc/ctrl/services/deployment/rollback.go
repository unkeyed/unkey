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

// Rollback switches traffic from the source deployment to a previous target
// deployment via a Restate workflow. The actual atomic mutation
// (route reassignment + apps.current_deployment_id update) is performed
// inside RoutingService.SwapLiveDeployment, which is per-env serialized.
// The workflow itself is keyed by source deployment_id.
func (s *Service) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	sourceID := req.Msg.GetSourceDeploymentId()
	targetID := req.Msg.GetTargetDeploymentId()
	if sourceID == "" || targetID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("source_deployment_id and target_deployment_id are required"))
	}
	if sourceID == targetID {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("source and target deployments must be different"))
	}

	// Validate here so callers get precise connect codes instead of
	// CodeInternal. The workflow re-checks everything except the target
	// status, environment, and desired_state gates, which exist only at
	// this layer.
	sourceDeployment, err := s.db.FindDeploymentById(ctx, sourceID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("source deployment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load source deployment: %w", err))
	}

	targetDeployment, err := s.db.FindDeploymentById(ctx, targetID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("target deployment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load target deployment: %w", err))
	}

	err = assert.All(
		assert.Equal(targetDeployment.ProjectID, sourceDeployment.ProjectID, "deployments must be in the same project"),
		assert.Equal(targetDeployment.AppID, sourceDeployment.AppID, "deployments must be in the same app"),
		assert.Equal(targetDeployment.EnvironmentID, sourceDeployment.EnvironmentID, "deployments must be in the same environment"),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}

	// The target starts serving live traffic the moment routes swap, so it
	// must have completed successfully and must not be draining toward
	// standby (a demoted deployment keeps status ready while it drains).
	if targetDeployment.Status != db.DeploymentsStatusReady {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("target deployment is not ready"))
	}
	if targetDeployment.DesiredState != db.DeploymentsDesiredStateRunning {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("target deployment is shutting down"))
	}

	environment, err := s.db.FindEnvironmentById(ctx, targetDeployment.EnvironmentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("environment not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load environment: %w", err))
	}
	// apps.current_deployment_id tracks the production live deployment;
	// rolling back outside production would corrupt that pointer.
	if environment.Slug != "production" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("only production deployments can be rolled back"))
	}

	app, err := s.db.FindAppById(ctx, sourceDeployment.AppID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load app: %w", err))
	}
	if !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String != sourceDeployment.ID {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("source deployment is not the current live deployment"))
	}

	logger.Info("initiating rollback via Restate",
		"source", req.Msg.GetSourceDeploymentId(),
		"target", req.Msg.GetTargetDeploymentId(),
	)

	_, err = s.deploymentClient(req.Msg.GetSourceDeploymentId()).
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
