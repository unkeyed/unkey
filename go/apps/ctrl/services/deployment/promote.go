package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

// Promote reassigns all domains to a deployment and removes the rolled back state
func (s *Service) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error) {

	s.logger.Info("initiating promotion",
		"target", req.Msg.GetTargetDeploymentId(),
	)

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

	project, err := db.Query.FindProjectById(ctx, s.db.RO(), targetDeployment.ProjectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", targetDeployment.ProjectID))
		}
		s.logger.Error("failed to get project",
			"project_id", targetDeployment.ProjectID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get project: %w", err))
	}

	if err := assert.All(
		assert.Equal(targetDeployment.Status, db.DeploymentsStatusReady),
		assert.True(project.LiveDeploymentID.Valid),
		assert.NotEqual(targetDeployment.ID, project.LiveDeploymentID.String),
	); err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
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

	domains, err := db.Query.FindDomainsForPromotion(ctx, s.db.RO(), db.FindDomainsForPromotionParams{
		EnvironmentID: sql.NullString{Valid: true, String: targetDeployment.EnvironmentID},
		Sticky: []db.NullDomainsSticky{
			db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyLive},
			db.NullDomainsSticky{Valid: true, DomainsSticky: db.DomainsStickyEnvironment},
		},
	})
	if err != nil {
		s.logger.Error("failed to get domains",
			"deployment_id", targetDeployment.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get domains: %w", err))
	}

	if len(domains) == 0 {
		s.logger.Error("no domains found",
			"deployment_id", targetDeployment.ID,
		)
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no domains found for deployment: %s", targetDeployment.ID))
	}

	gatewayConfig, err := pdb.Query.FindGatewayByDeploymentId(ctx, s.partitionDB.RO(), targetDeployment.ID)
	if err != nil {
		s.logger.Error("failed to get gateway config",
			"deployment_id", targetDeployment.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get gateway config: %w", err))
	}

	gatewayChanges := make([]pdb.UpsertGatewayParams, len(domains))
	for i, domain := range domains {
		gatewayChanges[i] = pdb.UpsertGatewayParams{
			WorkspaceID:  domain.WorkspaceID,
			DeploymentID: targetDeployment.ID,
			Hostname:     domain.Domain,
			Config:       gatewayConfig.Config,
		}
	}

	err = pdb.BulkQuery.UpsertGateway(ctx, s.partitionDB.RW(), gatewayChanges)
	if err != nil {
		s.logger.Error("failed to upsert gateway", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to upsert gateway: %w", err))
	}

	for _, domain := range domains {
		err = db.Query.ReassignDomain(ctx, s.db.RW(), db.ReassignDomainParams{
			ID:                domain.ID,
			TargetWorkspaceID: targetDeployment.WorkspaceID,
			DeploymentID:      sql.NullString{Valid: true, String: targetDeployment.ID},
			UpdatedAt:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			s.logger.Error("failed to update domain", "error", err.Error())
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update domain: %w", err))
		}
	}

	err = db.Query.UpdateProjectDeployments(ctx, s.db.RW(), db.UpdateProjectDeploymentsParams{
		ID:               project.ID,
		LiveDeploymentID: sql.NullString{Valid: true, String: targetDeployment.ID},
		IsRolledBack:     false,
		UpdatedAt:        sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		s.logger.Error("failed to update project deployments",
			"project_id", project.ID,
			"error", err.Error(),
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update project's live deployment id: %w", err))
	}

	return connect.NewResponse(&ctrlv1.PromoteResponse{}), nil
}
