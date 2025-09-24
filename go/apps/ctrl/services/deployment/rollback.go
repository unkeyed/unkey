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

	s.logger.Info("initiating rollback",
		"source", req.Msg.GetSourceDeploymentId(),
		"target", req.Msg.GetTargetDeploymentId(),
	)

	if err := assert.NotEqual(
		req.Msg.GetSourceDeploymentId(),
		req.Msg.GetTargetDeploymentId(),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	sourceDeployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetSourceDeploymentId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", req.Msg.GetSourceDeploymentId()))
		}
		s.logger.Error("failed to get deployment",
			"deployment_id", req.Msg.GetSourceDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	targetDeployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), req.Msg.GetTargetDeploymentId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment not found: %s", req.Msg.GetTargetDeploymentId()))
		}
		s.logger.Error("failed to get deployment",
			"deployment_id", req.Msg.GetTargetDeploymentId(),
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get deployment: %w", err))
	}

	if err := assert.All(
		assert.Equal(targetDeployment.EnvironmentID, sourceDeployment.EnvironmentID),
		assert.Equal(targetDeployment.ProjectID, sourceDeployment.ProjectID),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := db.Query.FindProjectById(ctx, s.db.RO(), sourceDeployment.ProjectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", sourceDeployment.ProjectID))
		}
		s.logger.Error("failed to get project",
			"project_id", sourceDeployment.ProjectID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get project: %w", err))
	}

	if err := assert.All(
		assert.True(project.LiveDeploymentID.Valid),
		assert.Equal(sourceDeployment.ID, project.LiveDeploymentID.String),
	); err != nil {
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
	domains, err := db.Query.FindDomainsByDeploymentId(ctx, s.db.RO(), sql.NullString{Valid: true, String: sourceDeployment.ID})
	if err != nil {
		s.logger.Error("failed to get domains",
			"deployment_id", sourceDeployment.ID,
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

	domainChanges := []db.ReassignDomainParams{}
	gatewayChanges := []pdb.UpsertGatewayParams{}

	for _, domain := range domains {
		if domain.Sticky.Valid &&
			(domain.Sticky.DomainsSticky == db.DomainsStickyLive ||
				domain.Sticky.DomainsSticky == db.DomainsStickyEnvironment) {

			domainChanges = append(domainChanges, db.ReassignDomainParams{
				ID:                 domain.ID,
				TargetWorkspaceID:  project.WorkspaceID,
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
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upsert gateway: %w", err))

	}

	// Not sure why there isn't a bulk query generated, but this will do for now
	// cause we're only rolling back one domain anyways
	for _, change := range domainChanges {
		err = db.Query.ReassignDomain(ctx, s.db.RW(), change)
		if err != nil {
			s.logger.Error("failed to update domain", "error", err.Error())
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update domain: %w", err))
		}
	}

	err = db.Query.UpdateProjectDeployments(ctx, s.db.RW(), db.UpdateProjectDeploymentsParams{
		ID:                     project.ID,
		LiveDeploymentID:       sql.NullString{Valid: true, String: targetDeployment.ID},
		RolledBackDeploymentID: sql.NullString{Valid: true, String: sourceDeployment.ID},
		UpdatedAt:              sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		s.logger.Error("failed to update project deployments",
			"project_id", project.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update project's live deployment id: %w", err))
	}

	res := &ctrlv1.RollbackResponse{
		Domains: make([]string, len(domainChanges)),
	}
	for i, domain := range domainChanges {
		res.Domains[i] = domain.ID
	}

	return connect.NewResponse(res), nil
}
