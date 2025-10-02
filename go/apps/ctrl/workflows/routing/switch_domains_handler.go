package routing

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

// SwitchDomains reassigns existing domains to a different deployment
func (s *Service) SwitchDomains(ctx restate.ObjectContext, req *hydrav1.SwitchDomainsRequest) (*hydrav1.SwitchDomainsResponse, error) {
	s.logger.Info("switching domains",
		"target_deployment_id", req.GetTargetDeploymentId(),
		"domain_count", len(req.GetDomainIds()),
	)

	// Fetch target deployment's gateway config
	gatewayConfig, err := restate.Run(ctx, func(stepCtx restate.RunContext) (partitiondb.FindGatewayByDeploymentIdRow, error) {
		return partitiondb.Query.FindGatewayByDeploymentId(stepCtx, s.partitionDB.RO(), req.GetTargetDeploymentId())
	}, restate.WithName("fetch-gateway-config"))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gateway config for deployment %s: %w", req.GetTargetDeploymentId(), err)
	}

	// Fetch domain info (to get hostnames and workspace_id)
	domains, err := restate.Run(ctx, func(stepCtx restate.RunContext) ([]db.FindDomainsByIdsRow, error) {
		return db.Query.FindDomainsByIds(stepCtx, s.db.RO(), req.GetDomainIds())
	}, restate.WithName("fetch-domains"))
	if err != nil {
		return nil, err
	}

	// Upsert gateway configs first
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		var gatewayParams []partitiondb.UpsertGatewayParams

		for _, domain := range domains {
			if isLocalHostname(domain.Domain, s.defaultDomain) {
				continue
			}

			gatewayParams = append(gatewayParams, partitiondb.UpsertGatewayParams{
				WorkspaceID:  domain.WorkspaceID,
				DeploymentID: req.GetTargetDeploymentId(),
				Hostname:     domain.Domain,
				Config:       gatewayConfig.Config,
			})
		}

		if len(gatewayParams) > 0 {
			if err := partitiondb.BulkQuery.UpsertGateway(stepCtx, s.partitionDB.RW(), gatewayParams); err != nil {
				return restate.Void{}, fmt.Errorf("failed to upsert gateway configs: %w", err)
			}
			s.logger.Info("updated gateway configs", "count", len(gatewayParams))
		}

		return restate.Void{}, nil
	}, restate.WithName("upsert-gateway-configs"))

	if err != nil {
		return nil, err
	}

	// Reassign domains
	_, err = restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		now := time.Now().UnixMilli()

		for _, domain := range domains {
			s.logger.Info("reassigning domain",
				"domain_id", domain.ID,
				"domain_name", domain.Domain,
			)

			err := db.Query.ReassignDomain(stepCtx, s.db.RW(), db.ReassignDomainParams{
				ID:                domain.ID,
				TargetWorkspaceID: domain.WorkspaceID,
				DeploymentID:      sql.NullString{Valid: true, String: req.GetTargetDeploymentId()},
				UpdatedAt:         sql.NullInt64{Valid: true, Int64: now},
			})
			if err != nil {
				return restate.Void{}, fmt.Errorf("failed to reassign domain %s: %w", domain.Domain, err)
			}
		}

		s.logger.Info("reassigned domains", "count", len(domains))
		return restate.Void{}, nil
	}, restate.WithName("reassign-domains"))

	if err != nil {
		return nil, err
	}

	s.logger.Info("domain switching completed",
		"target_deployment_id", req.GetTargetDeploymentId(),
		"domain_count", len(domains),
	)

	return &hydrav1.SwitchDomainsResponse{}, nil
}
