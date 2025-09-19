package deployment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

// Rollback performs a rollback to a previous deployment
// This is the main rollback implementation that the dashboard will call
func (s *Service) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error) {
	targetDeploymentID := req.Msg.GetTargetDeploymentId()
	projectID := req.Msg.GetProjectId()

	s.logger.Info("initiating rollback",
		"projectID", projectID,
		"targetDeploymentID", targetDeploymentID,
	)

	project, err := db.Query.FindProjectById(ctx, s.db.RO(), projectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", projectID))
		}
		s.logger.Error("failed to get project",
			"project_id", projectID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get project: %w", err))
	}

	if !project.LiveDeploymentID.Valid {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("project has no live deployment"))
	}

	// Get the target deployment and verify it belongs to the workspace
	liveDeployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), project.LiveDeploymentID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", targetDeploymentID))
		}
		s.logger.Error("failed to get deployment",
			"deployment_id", targetDeploymentID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	// Get the target deployment and verify it belongs to the workspace
	targetDeployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), targetDeploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", targetDeploymentID))
		}
		s.logger.Error("failed to get deployment",
			"deployment_id", targetDeploymentID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	err = assert.All(

		assert.Equal(liveDeployment.ProjectID, targetDeployment.ProjectID),
		assert.Equal(liveDeployment.EnvironmentID, targetDeployment.EnvironmentID),
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	vms, err := pdb.Query.FindVMsByDeploymentId(ctx, s.partitionDB.RO(), targetDeployment.ID)
	if err != nil {
		s.logger.Error("failed to get VMs",
			"deployment_id", targetDeployment.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get VMs: %w", err))
	}
	runningVms := 0
	for _, vm := range vms {
		if vm.Status == pdb.VmsStatusRunning {
			runningVms++
		}
	}
	if runningVms == 0 {
		s.logger.Error("no VMs found",
			"deployment_id", targetDeployment.ID,
		)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no VMs found for deployment: %s", targetDeployment.ID))
	}

	// get all domains on the live deployment that are sticky
	domains, err := db.Query.FindDomainsByDeploymentId(ctx, s.db.RO(), sql.NullString{Valid: true, String: liveDeployment.ID})
	if err != nil {
		s.logger.Error("failed to get domains",
			"deployment_id", liveDeployment.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get domains: %w", err))
	}

	gatewayConfig, err := pdb.Query.FindGatewayByDeploymentId(ctx, s.partitionDB.RO(), targetDeployment.ID)
	if err != nil {
		s.logger.Error("failed to get gateway config",
			"deployment_id", targetDeployment.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get gateway config: %w", err))
	}

	domainChanges := []db.RollBackDomainParams{}
	gatewayChanges := []pdb.UpsertGatewayParams{}

	for _, domain := range domains {
		if domain.Sticky.Valid &&
			(domain.Sticky.DomainsSticky == db.DomainsStickyLive ||
				domain.Sticky.DomainsSticky == db.DomainsStickyEnvironment) {

			domainChanges = append(domainChanges, db.RollBackDomainParams{
				ID:                 domain.ID,
				TargetDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
				IsRolledBack:       true,
				UpdatedAt:          sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})

			gatewayChanges = append(gatewayChanges, pdb.UpsertGatewayParams{
				WorkspaceID:  project.WorkspaceID,
				DeploymentID: targetDeployment.ID,
				Hostname:     domain.Domain,
				Config:       gatewayConfig.Config,
			})
		}

	}

	if len(domainChanges) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no domains to rollback"))
	}
	err = pdb.BulkQuery.UpsertGateway(ctx, s.partitionDB.RW(), gatewayChanges)
	if err != nil {
		s.logger.Error("failed to upsert gateway", "error", err.Error())
		return nil, fmt.Errorf("failed to upsert gateway: %w", err)
	}

	// Not sure why there isn't a bulk query generated, but this will do for now
	// cause we're only rolling back one domain anyways
	for _, change := range domainChanges {
		err = db.Query.RollBackDomain(ctx, s.db.RW(), change)
		if err != nil {
			s.logger.Error("failed to update domain", "error", err.Error())
			return nil, fmt.Errorf("failed to update domain: %w", err)
		}
	}

	err = db.Query.UpdateProjectLiveDeploymentId(ctx, s.db.RW(), db.UpdateProjectLiveDeploymentIdParams{
		ID:               project.ID,
		LiveDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
		UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		s.logger.Error("failed to update project active deployment ID",
			"project_id", project.ID,
			"error", err.Error(),
		)
		return nil, err
	}

	res := &ctrlv1.RollbackResponse{
		Domains: make([]string, len(domainChanges)),
	}
	for i, domain := range domainChanges {
		res.Domains[i] = domain.ID
	}

	return connect.NewResponse(res), nil
}
