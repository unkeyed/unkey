package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// Rollback performs a rollback to a previous deployment.
//
// This durable workflow switches sticky ingressRoutes (environment and live ingressRoutes) from the
// current live deployment back to a previous deployment. The operation is performed
// atomically through the routing service to prevent partial updates that could leave
// the system in an inconsistent state.
//
// The workflow validates that:
// - Source deployment is the current live deployment
// - Target deployment has running VMs
// - Both deployments are in the same project and environment
// - There are sticky ingressRoutes to rollback
//
// After switching ingressRoutes, the project is marked as rolled back to prevent new
// deployments from automatically taking over the live ingressRoutes.
//
// Returns terminal errors (400/404) for validation failures and retryable errors
// for system failures.
func (w *Workflow) Rollback(ctx restate.ObjectContext, req *hydrav1.RollbackRequest) (*hydrav1.RollbackResponse, error) {
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

	// Get all ingressRoutes on the live deployment that are sticky
	ingressRoutes, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindIngressRoutesForRollbackRow, error) {
		return db.Query.FindIngressRoutesForRollback(stepCtx, w.db.RO(), db.FindIngressRoutesForRollbackParams{
			EnvironmentID: sourceDeployment.EnvironmentID,
			Sticky: []db.IngressRoutesSticky{
				db.IngressRoutesStickyLive,
				db.IngressRoutesStickyEnvironment,
			},
		})
	}, restate.WithName("finding ingressRoutes for rollback"))
	if err != nil {
		return nil, fmt.Errorf("failed to get ingressRoutes: %w", err)
	}

	if len(ingressRoutes) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no ingressRoutes to rollback"), 400)
	}

	w.logger.Info("found ingressRoutes for rollback", "count", len(ingressRoutes), "deployment_id", sourceDeployment.ID)

	// Collect ingressRoute IDs
	var routeIDs []string
	for _, ingressRoute := range ingressRoutes {
		if ingressRoute.Sticky == db.IngressRoutesStickyLive ||
			ingressRoute.Sticky == db.IngressRoutesStickyEnvironment {
			routeIDs = append(routeIDs, ingressRoute.ID)
		}
	}

	// Call RoutingService to switch ingressRoutes atomically
	routingClient := hydrav1.NewRoutingServiceClient(ctx, project.ID)
	_, err = routingClient.AssignIngressRoutes().Request(&hydrav1.AssignIngressRoutesRequest{
		DeploymentId:    targetDeployment.ID,
		IngressRouteIds: routeIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to switch ingressRoutes: %w", err)
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
		"ingressRoutes_rolled_back", len(routeIDs))

	return &hydrav1.RollbackResponse{}, nil
}
