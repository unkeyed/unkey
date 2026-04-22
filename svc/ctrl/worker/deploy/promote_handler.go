package deploy

import (
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
	}, restate.WithName("finding target deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
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
	}, restate.WithName("finding app"), restate.WithMaxRetryAttempts(runMaxAttempts))
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

	// Resolve routes for normal promotion. Confirm-rollback skips this since
	// the routes already point at the target.
	var routeIDs []string
	if !isConfirmingRollback {
		frontlineRoutes, findErr := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindFrontlineRouteForPromotionRow, error) {
			return db.Query.FindFrontlineRouteForPromotion(stepCtx, w.db.RO(), db.FindFrontlineRouteForPromotionParams{
				EnvironmentID: targetDeployment.EnvironmentID,
				Sticky: []db.FrontlineRoutesSticky{
					db.FrontlineRoutesStickyLive,
					db.FrontlineRoutesStickyEnvironment,
				},
			})
		}, restate.WithName("finding frontlineRoutes for promotion"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if findErr != nil {
			return nil, fault.Wrap(findErr, fault.Public("Failed to find routes for promotion"))
		}

		if len(frontlineRoutes) == 0 {
			return nil, fault.Wrap(
				restate.TerminalError(fmt.Errorf("no frontline routes found for promotion"), 400),
				fault.Public("No routes found to promote"),
			)
		}

		logger.Info("found frontlineRoutes for promotion", "count", len(frontlineRoutes), "deployment_id", targetDeployment.ID)

		for _, route := range frontlineRoutes {
			routeIDs = append(routeIDs, route.ID)
		}
	}

	// Atomic swap inside the env-keyed Routing VO. For normal promotion this
	// reassigns the routes AND swaps current_deployment_id. For
	// confirm-rollback, routes are empty so it just clears is_rolled_back
	// (current_deployment_id is already the target, so the write is a no-op).
	swapResp, err := hydrav1.NewRoutingServiceClient(ctx, targetDeployment.EnvironmentID).
		SwapLiveDeployment().Request(&hydrav1.SwapLiveDeploymentRequest{
		DeploymentId:      targetDeployment.ID,
		FrontlineRouteIds: routeIDs,
		SetRollbackFlag:   false,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to swap live deployment"))
	}

	// Ensure the newly-promoted deployment is not spun down by any pending
	// standby schedule from when it was previously demoted.
	_, err = hydrav1.NewDeploymentServiceClient(ctx, targetDeployment.ID).ClearScheduledStateChanges().Request(&hydrav1.ClearScheduledStateChangesRequest{})
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to clear scheduled state changes on the promoted deployment"))
	}

	// Schedule the deployment that just got demoted for standby. For
	// normal promotion this is the previous current_deployment_id; for
	// confirm-rollback we look up the latest ready deployment that isn't
	// the target (the one that was rolled back from).
	var oldDeploymentID string
	if isConfirmingRollback {
		oldDeploymentID, err = restate.Run(ctx, func(stepCtx restate.RunContext) (string, error) {
			return db.Query.FindLatestReadyDeploymentByAppAndEnv(stepCtx, w.db.RO(), db.FindLatestReadyDeploymentByAppAndEnvParams{
				AppID:         targetDeployment.AppID,
				EnvironmentID: targetDeployment.EnvironmentID,
				ExcludeID:     targetDeployment.ID,
			})
		}, restate.WithName("finding old deployment to schedule for standby"), restate.WithMaxRetryAttempts(runMaxAttempts))
		if err != nil {
			return nil, fault.Wrap(err, fault.Public("Failed to find old deployment to schedule for standby"))
		}
	} else {
		oldDeploymentID = swapResp.GetPreviousDeploymentId()
	}

	if oldDeploymentID != "" {
		hydrav1.NewDeploymentServiceClient(ctx, oldDeploymentID).ScheduleDesiredStateChange().Send(&hydrav1.ScheduleDesiredStateChangeRequest{
			State:       hydrav1.DeploymentDesiredState_DEPLOYMENT_DESIRED_STATE_STANDBY,
			DelayMillis: (30 * time.Minute).Milliseconds(),
		})
		logger.Info("scheduled old deployment for standby", "old_deployment_id", oldDeploymentID)
	}

	logger.Info("promotion completed successfully",
		"target", req.GetTargetDeploymentId(),
		"domains_promoted", len(routeIDs),
		"confirm_rollback", isConfirmingRollback,
	)

	return &hydrav1.PromoteResponse{}, nil
}
