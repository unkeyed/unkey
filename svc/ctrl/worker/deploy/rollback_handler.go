package deploy

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/gatefault"
)

// Rollback performs a rollback to a previous deployment.
//
// This durable workflow switches sticky frontline routes (environment and live)
// from the current live deployment back to a previous deployment. The operation
// is performed atomically through the routing service to prevent partial updates
// that could leave the system in an inconsistent state.
//
// The workflow validates that source and target are different deployments, that
// the source deployment is the current live deployment, that both deployments
// belong to the same app and environment, and that there are sticky frontline
// routes to rollback.
//
// Before switching routes, any pending scheduled state changes on the target
// deployment are cleared so it won't be spun down while serving live traffic.
// After switching routes, the app is marked as rolled back to prevent new
// deployments from automatically taking over the live routes.
//
// Returns terminal errors (400/404) for validation failures and retryable errors
// for system failures.
func (w *Workflow) Rollback(ctx restate.ObjectContext, req *hydrav1.RollbackRequest) (*hydrav1.RollbackResponse, error) {
	logger.Info("initiating rollback",
		"source", req.GetSourceDeploymentId(),
		"target", req.GetTargetDeploymentId(),
	)

	if req.GetSourceDeploymentId() == req.GetTargetDeploymentId() {
		return nil, restate.TerminalError(fmt.Errorf("source and target deployments must be different"), 400)
	}

	sourceDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return w.db.FindDeploymentById(stepCtx, req.GetSourceDeploymentId())
	}, restate.WithName("finding source deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("source deployment not found: %s", req.GetSourceDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get source deployment: %w", err)
	}

	targetDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return w.db.FindDeploymentById(stepCtx, req.GetTargetDeploymentId())
	}, restate.WithName("finding target deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("target deployment not found: %s", req.GetTargetDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get target deployment: %w", err)
	}

	err = assert.All(
		assert.Equal(targetDeployment.ProjectID, sourceDeployment.ProjectID, "deployments must be in the same project"),
		assert.Equal(targetDeployment.AppID, sourceDeployment.AppID, "deployments must be in the same app"),
		assert.Equal(targetDeployment.EnvironmentID, sourceDeployment.EnvironmentID, "deployments must be in the same environment"),
	)
	if err != nil {
		return nil, restate.TerminalError(err, 400)
	}

	app, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.App, error) {
		return w.db.FindAppById(stepCtx, sourceDeployment.AppID)
	}, restate.WithName("finding app"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("app not found: %s", sourceDeployment.AppID), 404)
		}
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	if !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String != sourceDeployment.ID {
		return nil, restate.TerminalError(fmt.Errorf("source deployment is not the current deployment"), 400)
	}

	// Re-validate the target against the shared invariant at execution time (the
	// API and ctrl service already checked it) so a state change between enqueue
	// and execution fails the rollback instead of swapping traffic onto a target
	// that is no longer eligible. Environment is loaded only for its slug.
	environment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Environment, error) {
		return w.db.FindEnvironmentById(stepCtx, targetDeployment.EnvironmentID)
	}, restate.WithName("finding environment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("environment not found: %s", targetDeployment.EnvironmentID), 404)
		}
		return nil, fmt.Errorf("failed to get environment: %w", err)
	}

	if err := deploygate.CheckRollbackTarget(deploygate.RollbackInput{
		Status:              targetDeployment.Status,
		DesiredState:        targetDeployment.DesiredState,
		EnvironmentKind:     environment.Kind,
		CurrentDeploymentID: app.CurrentDeploymentID.String,
		DeploymentID:        targetDeployment.ID,
	}); err != nil {
		return nil, gatefault.Terminal(err)
	}

	// ensure the rolled back deployment does not get spun down from existing scheduled actions
	_, err = hydrav1.NewDeploymentServiceClient(ctx, targetDeployment.ID).ClearScheduledStateChanges().Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, err
	}

	frontlineRoutes, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindFrontlineRoutesForRollbackRow, error) {
		return w.db.FindFrontlineRoutesForRollback(stepCtx, db.FindFrontlineRoutesForRollbackParams{
			EnvironmentID: sourceDeployment.EnvironmentID,
			Sticky: []db.FrontlineRoutesSticky{
				db.FrontlineRoutesStickyLive,
				db.FrontlineRoutesStickyEnvironment,
			},
		})
	}, restate.WithName("finding frontlineRoutes for rollback"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("failed to get frontlineRoutes: %w", err)
	}

	if len(frontlineRoutes) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no frontlineRoutes to rollback"), 400)
	}

	logger.Info("found frontlineRoutes for rollback", "count", len(frontlineRoutes), "deployment_id", sourceDeployment.ID)

	var routeIDs []string
	for _, frontlineRoute := range frontlineRoutes {
		if frontlineRoute.Sticky == db.FrontlineRoutesStickyLive ||
			frontlineRoute.Sticky == db.FrontlineRoutesStickyEnvironment {
			routeIDs = append(routeIDs, frontlineRoute.ID)
		}
	}

	// Atomic swap inside the env-keyed Routing VO: reassign frontline routes
	// AND flip apps.current_deployment_id to the target with is_rolled_back=true.
	_, err = hydrav1.NewRoutingServiceClient(ctx, sourceDeployment.EnvironmentID).
		SwapLiveDeployment().Request(&hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:      targetDeployment.ID,
		FrontlineRouteIds: routeIDs,
		SetRollbackFlag:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to swap live deployment: %w", err)
	}

	logger.Info("updated app current deployment", "app_id", app.ID, "current_deployment_id", targetDeployment.ID)

	logger.Info("rollback completed successfully",
		"source", req.GetSourceDeploymentId(),
		"target", req.GetTargetDeploymentId(),
		"frontlineRoutes_rolled_back", len(routeIDs))

	if err := w.insertLifecycleAudit(
		ctx,
		req.GetActor(),
		req.GetCorrelationId(),
		targetDeployment,
		auditlog.DeploymentRollbackEvent,
		fmt.Sprintf("Rolled back to deployment %s", targetDeployment.ID),
	); err != nil {
		return nil, fmt.Errorf("insert rollback deployment audit log: %w", err)
	}

	return &hydrav1.RollbackResponse{}, nil
}
