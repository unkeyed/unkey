package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Rollback performs a rollback to a previous deployment.
//
// This durable workflow switches sticky frontlineRoutes (environment and live frontlineRoutes) from the
// current live deployment back to a previous deployment. The operation is performed
// atomically through the routing service to prevent partial updates that could leave
// the system in an inconsistent state.
//
// The workflow validates that:
// - Source deployment is the current live deployment
// - Target deployment has running VMs
// - Both deployments are in the same project and environment
// - There are sticky frontlineRoutes to rollback
//
// After switching frontlineRoutes, the project is marked as rolled back to prevent new
// deployments from automatically taking over the live frontlineRoutes.
//
// Returns terminal errors (400/404) for validation failures and retryable errors
// for system failures.
func (w *Workflow) Rollback(ctx restate.WorkflowSharedContext, req *hydrav1.RollbackRequest) (*hydrav1.RollbackResponse, error) {
	w.logger.Info("initiating rollback",
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

	// Get project
	project, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindProjectByIdRow, error) {
		return db.Query.FindProjectById(stepCtx, w.db.RO(), sourceDeployment.ProjectID)
	}, restate.WithName("finding project"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("project not found: %s", sourceDeployment.ProjectID), 404)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Validate source deployment is the live deployment
	if !project.LiveDeploymentID.Valid || project.LiveDeploymentID.String != sourceDeployment.ID {
		return nil, restate.TerminalError(fmt.Errorf("source deployment is not the current live deployment"), 400)
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

	w.logger.Info("found frontlineRoutes for rollback", "count", len(frontlineRoutes), "deployment_id", sourceDeployment.ID)

	// Collect frontlineRoute IDs
	var routeIDs []string
	for _, frontlineRoute := range frontlineRoutes {
		if frontlineRoute.Sticky == db.FrontlineRoutesStickyLive ||
			frontlineRoute.Sticky == db.FrontlineRoutesStickyEnvironment {
			routeIDs = append(routeIDs, frontlineRoute.ID)
		}
	}

	// Call RoutingService to switch frontlineRoutes atomically
	routingClient := hydrav1.NewRoutingServiceClient(ctx, project.ID)
	_, err = routingClient.AssignFrontlineRoutes().Request(&hydrav1.AssignFrontlineRoutesRequest{
		DeploymentId:      targetDeployment.ID,
		FrontlineRouteIds: routeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to switch frontlineRoutes: %w", err)
	}

	// Update project's live deployment
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err = db.Query.UpdateProjectDeployments(stepCtx, w.db.RW(), db.UpdateProjectDeploymentsParams{
			ID:               project.ID,
			LiveDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
			IsRolledBack:     true,
			UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return restate.Void{}, fmt.Errorf("failed to update project's live deployment id: %w", err)
		}
		w.logger.Info("updated project live deployment", "project_id", project.ID, "live_deployment_id", targetDeployment.ID)
		return restate.Void{}, nil
	}, restate.WithName("updating project live deployment"))
	if err != nil {
		return nil, err
	}

	w.logger.Info("rollback completed successfully",
		"source", req.GetSourceDeploymentId(),
		"target", req.GetTargetDeploymentId(),
		"frontlineRoutes_rolled_back", len(routeIDs))

	return &hydrav1.RollbackResponse{}, nil
}
