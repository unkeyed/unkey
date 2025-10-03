package routing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

// AssignDomains creates or reassigns domains to a deployment and creates gateway configs.
//
// This durable workflow performs the following steps for each domain:
// 1. Check if domain exists in the database
// 2. If new, create domain record with specified sticky behavior
// 3. If existing and not rolled back, reassign to new deployment
// 4. If existing and rolled back, skip reassignment
// 5. Create gateway configs for all changed domains (except local hostnames)
//
// Each domain upsert is wrapped in a separate restate.Run call with a unique name,
// allowing partial completion tracking. If the workflow fails after creating some domains,
// Restate will skip the already-created domains on retry.
//
// Gateway configs are updated in bulk for all changed domains, using protojson encoding
// for easier debugging. Local hostnames (localhost, *.local, *.test) are skipped to
// prevent unnecessary config creation during local development.
//
// Returns the list of domain names that were actually modified (created or reassigned).
func (s *Service) AssignDomains(ctx restate.ObjectContext, req *hydrav1.AssignDomainsRequest) (*hydrav1.AssignDomainsResponse, error) {
	s.logger.Info("assigning domains",
		"deployment_id", req.GetDeploymentId(),
		"domain_count", len(req.GetDomains()),
	)

	changedDomains := []string{}

	// Upsert each domain in the database
	for _, domain := range req.GetDomains() {
		changed, err := restate.Run(ctx, func(stepCtx restate.RunContext) (bool, error) {
			now := time.Now().UnixMilli()

			var wasChanged bool
			err := db.Tx(stepCtx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
				existing, err := db.Query.FindDomainByDomain(txCtx, tx, domain.GetName())
				if err != nil {
					if !db.IsNotFound(err) {
						return fmt.Errorf("failed to find domain: %w", err)
					}

					// Domain does not exist, create it
					sticky := parseDomainSticky(domain.GetSticky())
					err := db.Query.InsertDomain(txCtx, tx, db.InsertDomainParams{
						ID:            uid.New("domain"),
						WorkspaceID:   req.GetWorkspaceId(),
						ProjectID:     sql.NullString{Valid: true, String: req.GetProjectId()},
						EnvironmentID: sql.NullString{Valid: true, String: req.GetEnvironmentId()},
						Domain:        domain.GetName(),
						Sticky:        sticky,
						DeploymentID:  sql.NullString{Valid: true, String: req.GetDeploymentId()},
						CreatedAt:     now,
						Type:          db.DomainsTypeWildcard,
					})
					if err != nil {
						return fmt.Errorf("failed to insert domain: %w", err)
					}
					wasChanged = true
					return nil
				}

				// Domain exists
				if req.GetIsRolledBack() {
					s.logger.Info("skipping domain assignment - project is rolled back",
						"domain_id", existing.ID,
						"domain", existing.Domain,
					)
					return nil
				}

				// Reassign domain to new deployment
				err = db.Query.ReassignDomain(txCtx, tx, db.ReassignDomainParams{
					ID:                existing.ID,
					TargetWorkspaceID: req.GetWorkspaceId(),
					DeploymentID:      sql.NullString{Valid: true, String: req.GetDeploymentId()},
					UpdatedAt:         sql.NullInt64{Valid: true, Int64: now},
				})
				if err != nil {
					return fmt.Errorf("failed to reassign domain: %w", err)
				}
				wasChanged = true
				return nil
			})

			return wasChanged, err
		}, restate.WithName(fmt.Sprintf("upsert-domain-%s", domain.GetName())))

		if err != nil {
			return nil, err
		}

		if changed {
			changedDomains = append(changedDomains, domain.GetName())
		}
	}

	// Create gateway configs for changed domains (except local ones)
	_, err := restate.Run(ctx, func(stepCtx restate.RunContext) (restate.Void, error) {
		var gatewayParams []partitiondb.UpsertGatewayParams
		var skippedDomains []string

		for _, domainName := range changedDomains {
			if isLocalHostname(domainName, s.defaultDomain) {
				skippedDomains = append(skippedDomains, domainName)
				continue
			}

			// Marshal gateway config to JSON
			configBytes, err := protojson.Marshal(req.GetGatewayConfig())
			if err != nil {
				s.logger.Error("failed to marshal gateway config", "error", err, "domain", domainName)
				continue
			}

			gatewayParams = append(gatewayParams, partitiondb.UpsertGatewayParams{
				WorkspaceID:  req.GetWorkspaceId(),
				DeploymentID: req.GetDeploymentId(),
				Hostname:     domainName,
				Config:       configBytes,
			})
		}

		// Bulk upsert gateway configs
		if len(gatewayParams) > 0 {
			if err := partitiondb.BulkQuery.UpsertGateway(stepCtx, s.partitionDB.RW(), gatewayParams); err != nil {
				return restate.Void{}, fmt.Errorf("failed to upsert gateway configs: %w", err)
			}
			s.logger.Info("created gateway configs",
				"count", len(gatewayParams),
				"skipped", len(skippedDomains),
			)
		}

		return restate.Void{}, nil
	}, restate.WithName("create-gateway-configs"))

	if err != nil {
		return nil, err
	}

	s.logger.Info("domain assignment completed",
		"deployment_id", req.GetDeploymentId(),
		"changed_domains", len(changedDomains),
	)

	return &hydrav1.AssignDomainsResponse{
		ChangedDomainNames: changedDomains,
	}, nil
}
