package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
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
			return nil, fault.Wrap(
				restate.TerminalError(fmt.Errorf("deployment not found: %s", req.GetTargetDeploymentId()), 404),
				fault.Public("The deployment could not be found"),
			)
		}
		return nil, fault.Wrap(err, fault.Public("Failed to find the target deployment"))
	}

	// Get app from deployment's app_id
	app, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.App, error) {
		return db.Query.FindAppById(stepCtx, w.db.RO(), targetDeployment.AppID)
	}, restate.WithName("finding app"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.Wrap(
				restate.TerminalError(fmt.Errorf("app not found: %s", targetDeployment.AppID), 404),
				fault.Public("The project could not be found"),
			)
		}
		return nil, fault.Wrap(err, fault.Public("Failed to find the app"))
	}

	// Validate preconditions
	if targetDeployment.Status != db.DeploymentsStatusReady {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("deployment status must be ready, got: %s", targetDeployment.Status), 400),
			fault.Public("The deployment is not ready for promotion"),
		)
	}
	if !app.CurrentDeploymentID.Valid {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("app has no live deployment"), 400),
			fault.Public("The app has no live deployment to promote from"),
		)
	}
	if targetDeployment.ID == app.CurrentDeploymentID.String {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("target deployment is already the live deployment"), 400),
			fault.Public("This deployment is already live"),
		)
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
		return nil, fault.Wrap(err, fault.Public("Failed to find routes for promotion"))
	}

	if len(frontlineRoutes) == 0 {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("no frontline routes found for promotion"), 400),
			fault.Public("No routes found to promote"),
		)
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
		return nil, fault.Wrap(err, fault.Public("Failed to switch routes to the promoted deployment"))
	}

	// Update app's current deployment
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err = db.Query.UpdateAppDeployments(stepCtx, w.db.RW(), db.UpdateAppDeploymentsParams{
			AppID:               app.ID,
			CurrentDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
			IsRolledBack:        false,
			UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return restate.Void{}, fault.Wrap(err, fault.Internal("failed to update app's current deployment id"))
		}
		logger.Info("updated app current deployment", "app_id", app.ID, "current_deployment_id", targetDeployment.ID)
		return restate.Void{}, nil
	}, restate.WithName("updating app current deployment"))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to update the project after promotion"))
	}

	// ensure the new promoted deployment does not get spun down from existing scheduled actions
	_, err = hydrav1.NewDeploymentServiceClient(ctx, targetDeployment.ID).ClearScheduledStateChanges().Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to clear scheduled state changes on the promoted deployment"))
	}

	// schedule old deployment to be spun down
	hydrav1.NewDeploymentServiceClient(ctx, app.CurrentDeploymentID.String).ScheduleDesiredStateChange().Send(&hydrav1.ScheduleDesiredStateChangeRequest{
		State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
		DelayMillis: (30 * time.Minute).Milliseconds(),
	})

	logger.Info("promotion completed successfully",
		"target", req.GetTargetDeploymentId(),
		"domains_promoted", len(routeIDs))

	return &hydrav1.PromoteResponse{}, nil
}
