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

// Promote reassigns all sticky domains to a deployment and clears the rolled back state.
//
// This durable workflow moves sticky domains (environment and live) from the
// current live deployment to a new target deployment. It reverses a previous
// rollback and allows normal deployment flow to resume.
//
// The workflow validates that the target deployment is ready, the app has a
// live deployment, the target is not already the live deployment, and there are
// sticky domains to promote.
//
// After switching domains atomically through the routing service, the app's live
// deployment pointer is updated and the rolled back flag is cleared, allowing future
// deployments to automatically take over sticky domains. Any pending scheduled
// state changes on the promoted deployment are cleared (so it won't be spun down),
// and the previous live deployment is scheduled for standby after 30 minutes.
//
// Returns terminal errors (400/404) for validation failures and retryable errors
// for system failures.
func (w *Workflow) Promote(ctx restate.WorkflowSharedContext, req *hydrav1.PromoteRequest) (*hydrav1.PromoteResponse, error) {
	logger.Info("initiating promotion", "target", req.GetTargetDeploymentId())

	// Get target deployment
	targetDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.Deployment, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RO(), req.GetTargetDeploymentId())
	}, restate.WithName("finding target deployment"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("deployment not found: %s", req.GetTargetDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get target deployment: %w", err)
	}

	// Get app from deployment's app_id
	app, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.App, error) {
		row, err := db.Query.FindAppById(stepCtx, w.db.RO(), targetDeployment.AppID)
		if err != nil {
			return db.App{}, err
		}
		return row.App, nil
	}, restate.WithName("finding app"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("app not found: %s", targetDeployment.AppID), 404)
		}
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Validate preconditions
	if targetDeployment.Status != db.DeploymentsStatusReady {
		return nil, restate.TerminalError(fmt.Errorf("deployment status must be ready, got: %s", targetDeployment.Status), 400)
	}
	if !app.LiveDeploymentID.Valid {
		return nil, restate.TerminalError(fmt.Errorf("app has no live deployment"), 400)
	}
	if targetDeployment.ID == app.LiveDeploymentID.String {
		return nil, restate.TerminalError(fmt.Errorf("target deployment is already the live deployment"), 400)
	}

	// Get all frontlineRoutes for promotion
	frontlineRoutes, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindFrontlineRouteForPromotionRow, error) {
		return db.Query.FindFrontlineRouteForPromotion(stepCtx, w.db.RO(), db.FindFrontlineRouteForPromotionParams{
			EnvironmentID: targetDeployment.EnvironmentID,
			Sticky: []db.FrontlineRoutesSticky{
				db.FrontlineRoutesStickyLive,
				db.FrontlineRoutesStickyEnvironment,
			},
		})
	}, restate.WithName("finding frontlineRoutes for promotion"))
	if err != nil {
		return nil, fmt.Errorf("failed to get frontlineRoutes: %w", err)
	}

	if len(frontlineRoutes) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no frontlineRoutes found for promotion"), 400)
	}

	logger.Info("found frontlineRoutes for promotion", "count", len(frontlineRoutes), "deployment_id", targetDeployment.ID)

	// Collect domain IDs
	var routeIDs []string
	for _, route := range frontlineRoutes {
		routeIDs = append(routeIDs, route.ID)
	}

	// Call RoutingService to switch routes atomically, keyed by app ID
	routingClient := hydrav1.NewRoutingServiceClient(ctx, app.ID)
	_, err = routingClient.AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      targetDeployment.ID,
		FrontlineRouteIds: routeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to switch domains: %w", err)
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

	// ensure the new promoted deployment does not get spun down from existing scheduled actions
	_, err = hydrav1.NewDeploymentServiceClient(ctx, targetDeployment.ID).ClearScheduledStateChanges().Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, err
	}

	// schedule old deployment to be spun down
	hydrav1.NewDeploymentServiceClient(ctx, app.LiveDeploymentID.String).ScheduleDesiredStateChange().Send(&hydrav1.ScheduleDesiredStateChangeRequest{
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
		DelayMillis: (30 * time.Minute).Milliseconds(),
	})

	logger.Info("promotion completed successfully",
		"target", req.GetTargetDeploymentId(),
		"domains_promoted", len(routeIDs))

	return &hydrav1.PromoteResponse{}, nil
}
