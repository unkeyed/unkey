package deploy

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// Promote reassigns all sticky domains to a deployment and clears the rolled back state.
//
// This durable workflow moves sticky domains (environment and live domains) from the
// current live deployment to a new target deployment. It reverses a previous rollback
// and allows normal deployment flow to resume.
//
// The workflow validates that:
// - Target deployment is ready (not building, deploying, or failed)
// - Target deployment has running VMs
// - Target deployment is not already the live deployment
// - Project has sticky domains to promote
//
// After switching domains atomically through the routing service, the project's live
// deployment pointer is updated and the rolled back flag is cleared, allowing future
// deployments to automatically take over sticky domains.
//
// Returns terminal errors (400/404) for validation failures and retryable errors
// for system failures.
func (w *Workflow) Promote(ctx restate.ObjectContext, req *hydrav1.PromoteRequest) (*hydrav1.PromoteResponse, error) {
	w.logger.Info("initiating promotion", "target", req.GetTargetDeploymentId())

	// Get target deployment
	targetDeployment, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindDeploymentByIdRow, error) {
		return db.Query.FindDeploymentById(stepCtx, w.db.RO(), req.GetTargetDeploymentId())
	}, restate.WithName("finding target deployment"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("deployment not found: %s", req.GetTargetDeploymentId()), 404)
		}
		return nil, fmt.Errorf("failed to get target deployment: %w", err)
	}

	// Get project
	project, err := restate.Run(ctx, func(stepCtx restate.RunContext) (db.FindProjectByIdRow, error) {
		return db.Query.FindProjectById(stepCtx, w.db.RO(), targetDeployment.ProjectID)
	}, restate.WithName("finding project"))
	if err != nil {
		if db.IsNotFound(err) {
			return nil, restate.TerminalError(fmt.Errorf("project not found: %s", targetDeployment.ProjectID), 404)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Validate preconditions
	if targetDeployment.Status != db.DeploymentsStatusReady {
		return nil, restate.TerminalError(fmt.Errorf("deployment status must be ready, got: %s", targetDeployment.Status), 400)
	}
	if !project.LiveDeploymentID.Valid {
		return nil, restate.TerminalError(fmt.Errorf("project has no live deployment"), 400)
	}
	if targetDeployment.ID == project.LiveDeploymentID.String {
		return nil, restate.TerminalError(fmt.Errorf("target deployment is already the live deployment"), 400)
	}

	// Check target deployment has running VMs
	vms, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.Vm, error) {
		return db.Query.FindVMsByDeploymentId(stepCtx, w.db.RO(), targetDeployment.ID)
	}, restate.WithName("finding target VMs"))
	if err != nil {
		return nil, fmt.Errorf("failed to get VMs: %w", err)
	}

	runningVms := 0
	for _, vm := range vms {
		if vm.Status == db.VmsStatusRunning {
			runningVms++
		}
	}
	if runningVms == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no running VMs found for target deployment: %s", targetDeployment.ID), 400)
	}

	w.logger.Info("found running VMs for target deployment", "count", runningVms, "deployment_id", targetDeployment.ID)

	// Get all domains for promotion
	domains, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindDomainsForPromotionRow, error) {
		return db.Query.FindDomainsForPromotion(stepCtx, w.db.RO(), db.FindDomainsForPromotionParams{
			EnvironmentID: sql.NullString{Valid: true, String: targetDeployment.EnvironmentID},
			Sticky: []db.NullDomainsSticky{
				{Valid: true, DomainsSticky: db.DomainsStickyLive},
				{Valid: true, DomainsSticky: db.DomainsStickyEnvironment},
			},
		})
	}, restate.WithName("finding domains for promotion"))
	if err != nil {
		return nil, fmt.Errorf("failed to get domains: %w", err)
	}

	if len(domains) == 0 {
		return nil, restate.TerminalError(fmt.Errorf("no domains found for promotion"), 400)
	}

	w.logger.Info("found domains for promotion", "count", len(domains), "deployment_id", targetDeployment.ID)

	// Collect domain IDs
	var domainIDs []string
	for _, domain := range domains {
		domainIDs = append(domainIDs, domain.ID)
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

	// Update project's live deployment and clear rolled back flag
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		err = db.Query.UpdateProjectDeployments(stepCtx, w.db.RW(), db.UpdateProjectDeploymentsParams{
			ID:               project.ID,
			LiveDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
			IsRolledBack:     false,
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

	w.logger.Info("promotion completed successfully",
		"target", req.GetTargetDeploymentId(),
		"domains_promoted", len(domainIDs))

	return &hydrav1.PromoteResponse{}, nil
}
