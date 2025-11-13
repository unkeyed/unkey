package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

// Rollback performs a rollback to a previous deployment.
//
// This durable workflow switches sticky domains (environment and live domains) from the
// current live deployment back to a previous deployment. The operation is performed
// atomically through the routing service to prevent partial updates that could leave
// the system in an inconsistent state.
//
// The workflow validates that:
// - Source deployment is the current live deployment
// - Target deployment has running VMs
// - Both deployments are in the same project and environment
// - There are sticky domains to rollback
//
// After switching domains, the project is marked as rolled back to prevent new
// deployments from automatically taking over the live domains.
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
	sourceDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindDeploymentByIdRow, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RO(), req.GetSourceDeploymentId())
	}, restate.WithName("finding source deployment"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("source deployment not found: %s", req.GetSourceDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get source deployment: %w", err)
	}

	// Get target deployment
	targetDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindDeploymentByIdRow, error) {
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

	// Check target deployment has running instances
	instances, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]partitiondb.Instance, error) {
		return partitiondb.Query.FindInstancesByDeploymentId(stepCtx, w.partitionDB.RO(), targetDeployment.ID)
	}, restate.WithName("finding target instances"))
	if err != nil {
		return nil, fmt.Errorf("failed to get instances: %w", err)
	}

	runningInstances := 0
	for _, instance := range instances {
		if instance.Status == partitiondb.InstanceStatusRunning {
			runningInstances++
		}
	}
	if runningInstances == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no running instances found for target deployment: %s", targetDeployment.ID), 400)
	}

	w.logger.Info("found running instances for target deployment", "count", runningInstances, "deployment_id", targetDeployment.ID)

	// Get all domains on the live deployment that are sticky
	domains, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindDomainsForRollbackRow, error) {
		return db.Query.FindDomainsForRollback(stepCtx, w.db.RO(), db.FindDomainsForRollbackParams{
			EnvironmentID: sql.NullString{Valid: true, String: sourceDeployment.EnvironmentID},
			Sticky: []db.NullDomainsSticky{
				{Valid: true, DomainsSticky: db.DomainsStickyLive},
				{Valid: true, DomainsSticky: db.DomainsStickyEnvironment},
			},
		})
	}, restate.WithName("finding domains for rollback"))
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}

	if len(domains) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no domains to rollback"), 400)
	}

	w.logger.Info("found domains for rollback", "count", len(domains), "deployment_id", sourceDeployment.ID)

	// Collect domain IDs
	var domainIDs []string
	for _, domain := range domains {
		if domain.Sticky.Valid &&
			(domain.Sticky.DomainsSticky == db.DomainsStickyLive ||
				domain.Sticky.DomainsSticky == db.DomainsStickyEnvironment) {
			domainIDs = append(domainIDs, domain.ID)
		}
	}

	// Call RoutingService to switch domains atomically
	routingClient := hydrav1.NewRoutingServiceClient(ctx, project.ID)
	_, err = routingClient.SwitchDomains().Request(&hydrav1.SwitchDomainsRequest{
		TargetDeploymentId: targetDeployment.ID,
		DomainIds:          domainIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to switch domains: %w", err)
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
		"domains_rolled_back", len(domainIDs))

	return &hydrav1.RollbackResponse{}, nil
}
