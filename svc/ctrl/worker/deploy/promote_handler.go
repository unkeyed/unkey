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
// This durable workflow supports two modes:
//
// 1. Normal promotion: moves sticky domains (environment and live) from the
// current live deployment to a new target deployment.
//
// 2. Confirm rollback: when the app is rolled back and the target is already
// the current deployment, clears the rolled back flag without reassigning
// routes (they already point to the correct deployment). This allows future
// deployments to automatically take over sticky domains again. The deployment
// that was rolled back from is scheduled for standby after 30 minutes.
//
// The workflow validates that the target deployment is ready and the app has a
// live deployment. For normal promotion, it also validates that the target is
// not already the live deployment and that there are sticky domains to promote.
//
// Returns terminal errors (400/404) for validation failures and retryable errors
// for system failures.
func (w *Workflow) Promote(ctx restate.ObjectContext, req *hydrav1.PromoteRequest) (*hydrav1.PromoteResponse, error) {
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
	isConfirmingRollback := app.IsRolledBack && targetDeployment.ID == app.CurrentDeploymentID.String
	// This guards against us forcing current deployment to promotion
	if targetDeployment.ID == app.CurrentDeploymentID.String && !app.IsRolledBack {
		return nil, fault.Wrap(
			restate.TerminalError(fmt.Errorf("target deployment is already the live deployment"), 400),
			fault.Public("This deployment is already live"),
		)
	}

	if isConfirmingRollback {
		logger.Info("confirming rollback", "deployment_id", targetDeployment.ID, "app_id", app.ID)

		// Clear scheduled state changes on the current deployment
		_, err = hydrav1.NewDeploymentServiceClient(ctx, targetDeployment.ID).ClearScheduledStateChanges().Request(&hydrav1.ClearScheduledStateChangesRequest{})
		if err != nil {
			return nil, fault.Wrap(err, fault.Public("Failed to clear scheduled state changes on the current deployment"))
		}

		// Clear isRolledBack flag, routes already point to the correct deployment
		_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
			err = db.Query.UpdateAppDeployments(stepCtx, w.db.RW(), db.UpdateAppDeploymentsParams{
				AppID:               app.ID,
				CurrentDeploymentID: app.CurrentDeploymentID,
				IsRolledBack:        false,
				UpdatedAt:           sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return restate.Void{}, fault.Wrap(err, fault.Internal("failed to clear isRolledBack flag"))
			}
			return restate.Void{}, nil
		}, restate.WithName("clearing isRolledBack flag"))
		if err != nil {
			return nil, fault.Wrap(err, fault.Public("Failed to confirm rollback"))
		}

		// Find the deployment that was rolled back from and schedule it for standby
		oldDeploymentID, findErr := restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			return db.Query.FindLatestReadyDeploymentByAppAndEnv(stepCtx, w.db.RO(), db.FindLatestReadyDeploymentByAppAndEnvParams{
				AppID:         targetDeployment.AppID,
				EnvironmentID: targetDeployment.EnvironmentID,
				ExcludeID:     targetDeployment.ID,
			})
		}, restate.WithName("finding old deployment to schedule for standby"))

		if findErr != nil {
			logger.Error("failed to find old deployment to schedule for standby",
				"app_id", targetDeployment.AppID,
				"environment_id", targetDeployment.EnvironmentID,
				"error", findErr,
			)
		} else if oldDeploymentID != "" {
			hydrav1.NewDeploymentServiceClient(ctx, oldDeploymentID).ScheduleDesiredStateChange().Send(&hydrav1.ScheduleDesiredStateChangeRequest{
				State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
				DelayMillis: (30 * time.Minute).Milliseconds(),
			})
			logger.Info("scheduled old deployment for standby", "old_deployment_id", oldDeploymentID)
		}

		logger.Info("rollback confirmed successfully", "deployment_id", targetDeployment.ID)
		return &hydrav1.PromoteResponse{}, nil
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
