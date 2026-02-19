package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
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
func (w *Workflow) Rollback(ctx restate.WorkflowSharedContext, req *hydrav1.RollbackRequest) (*hydrav1.RollbackResponse, error) {
	logger.Info("initiating rollback",
		"source", req.GetSourceDeploymentId(),
		"target", req.GetTargetDeploymentId(),
	)

	// Validate source and target are different
	if req.GetSourceDeploymentId() == req.GetTargetDeploymentId() {
		return nil, restate.TerminalError(fmt.Errorf("source and target deployments must be different"), 400)
	}

	// Get source deployment
	sourceDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RO(), req.GetSourceDeploymentId())
	}, restate.WithName("finding source deployment"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("source deployment not found: %s", req.GetSourceDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get source deployment: %w", err)
	}

	// Get target deployment
	targetDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RO(), req.GetTargetDeploymentId())
	}, restate.WithName("finding target deployment"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("target deployment not found: %s", req.GetTargetDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get target deployment: %w", err)
	}

	// Validate deployments are in same environment and project
	if targetDeployment.EnvironmentID != sourceDeployment.EnvironmentID {
		return nil, restate.TerminalError(fmt.Errorf("deployments must be in the same environment"), 400)
	}

	if targetDeployment.ProjectID != sourceDeployment.ProjectID {
		return nil, restate.TerminalError(fmt.Errorf("deployments must be in the same project"), 400)
	}

	if targetDeployment.AppID != sourceDeployment.AppID {
		return nil, restate.TerminalError(fmt.Errorf("deployments must be in the same app"), 400)
	}

	// Get app from deployment's app_id
	app, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.App, error) {
		row, err := db.Query.FindAppById(stepCtx, w.db.RO(), sourceDeployment.AppID)
		if err != nil {
			return db.App{}, err
		}
		return row.App, nil
	}, restate.WithName("finding app"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("app not found: %s", sourceDeployment.AppID), 404)
		}
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Validate source deployment is the live deployment
	if !app.LiveDeploymentID.Valid || app.LiveDeploymentID.String != sourceDeployment.ID {
		return nil, restate.TerminalError(fmt.Errorf("source deployment is not the current live deployment"), 400)
	}

	// ensure the rolled back deployment does not get spun down from existing scheduled actions
	_, err = hydrav1.NewDeploymentServiceClient(ctx, targetDeployment.ID).ClearScheduledStateChanges().Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, err
	}

	// Get all frontlineRoutes on the live deployment that are sticky
	frontlineRoutes, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindFrontlineRoutesForRollbackRow, error) {
		return db.Query.FindFrontlineRoutesForRollback(stepCtx, w.db.RO(), db.FindFrontlineRoutesForRollbackParams{
			EnvironmentID: sourceDeployment.EnvironmentID,
			Sticky: []db.FrontlineRoutesSticky{
				db.FrontlineRoutesStickyLive,
				db.FrontlineRoutesStickyEnvironment,
			},
		})
	}, restate.WithName("finding frontlineRoutes for rollback"))
	if err != nil {
		return nil, fmt.Errorf("failed to get frontlineRoutes: %w", err)
	}

	if len(frontlineRoutes) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no frontlineRoutes to rollback"), 400)
	}

	logger.Info("found frontlineRoutes for rollback", "count", len(frontlineRoutes), "deployment_id", sourceDeployment.ID)

	// Collect frontlineRoute IDs
	var routeIDs []string
	for _, frontlineRoute := range frontlineRoutes {
		if frontlineRoute.Sticky == db.FrontlineRoutesStickyLive ||
			frontlineRoute.Sticky == db.FrontlineRoutesStickyEnvironment {
			routeIDs = append(routeIDs, frontlineRoute.ID)
		}
	}

	// Call RoutingService to switch frontlineRoutes atomically, keyed by app ID
	routingClient := hydrav1.NewRoutingServiceClient(ctx, app.ID)
	_, err = routingClient.AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      targetDeployment.ID,
		FrontlineRouteIds: routeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to switch frontlineRoutes: %w", err)
	}

	// Update app's live deployment
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err = db.Query.UpdateAppDeployments(stepCtx, w.db.RW(), db.UpdateAppDeploymentsParams{
			ID:               app.ID,
			LiveDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
			UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return restate.Void{}, fmt.Errorf("failed to update app's live deployment id: %w", err)
		}
		logger.Info("updated app live deployment", "app_id", app.ID, "live_deployment_id", targetDeployment.ID)
		return restate.Void{}, nil
	}, restate.WithName("updating app live deployment"))
	if err != nil {
		return nil, err
	}

	logger.Info("rollback completed successfully",
		"source", req.GetSourceDeploymentId(),
		"target", req.GetTargetDeploymentId(),
		"frontlineRoutes_rolled_back", len(routeIDs))

	return &hydrav1.RollbackResponse{}, nil
}
