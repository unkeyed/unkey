package certificate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

// InfraWorkspaceID is the workspace ID used for infrastructure certificates.
const InfraWorkspaceID = "unkey_internal"

// BootstrapConfig holds configuration for infrastructure certificate bootstrapping.
type BootstrapConfig struct {
	// DefaultDomain is the base domain for deployments (e.g., "unkey.app", "unkey.fun").
	// Results in a wildcard cert for "*.unkey.app".
	DefaultDomain string

	// ApexDomain is the base domain for cross-region frontline communication (e.g., "unkey.cloud").
	ApexDomain string

	// Regions is the list of available regions (e.g., ["us-west-2.aws", "eu-central-1.aws"]).
	// Combined with ApexDomain to create certs like "*.us-west-2.aws.unkey.cloud".
	Regions []string

	// RestateClient is used to invoke the ProcessChallenge workflow.
	RestateClient hydrav1.CertificateServiceIngressClient
}

// BootstrapInfraCerts ensures infrastructure wildcard certificates are provisioned.
//
// This creates custom_domain and acme_challenge records for each infrastructure domain,
// then triggers the existing ProcessChallenge Restate workflow to obtain the certs.
//
// Handles:
//   - Default domain wildcard (e.g., "*.unkey.app") for deployment TLS
//   - Per-region wildcards (e.g., "*.us-west-2.aws.unkey.cloud") for cross-region frontline
//
// Idempotent - skips domains that already have certs or pending challenges.
func (s *Service) BootstrapInfraCerts(ctx context.Context, cfg BootstrapConfig) error {
	if s.dnsProvider == nil {
		s.logger.Warn("DNS provider not configured, skipping infrastructure cert bootstrap")
		return nil
	}

	// Collect all wildcard domains we need
	var domains []string

	// Default domain wildcard (e.g., *.unkey.app)
	if cfg.DefaultDomain != "" {
		domains = append(domains, "*."+cfg.DefaultDomain)
	}

	// Per-region wildcards (e.g., *.us-west-2.aws.unkey.cloud)
	if cfg.ApexDomain != "" {
		for _, region := range cfg.Regions {
			domains = append(domains, fmt.Sprintf("*.%s.%s", region, cfg.ApexDomain))
		}
	}

	if len(domains) == 0 {
		s.logger.Info("no infrastructure domains configured, skipping cert bootstrap")
		return nil
	}

	for _, domain := range domains {
		if err := s.ensureInfraDomain(ctx, domain, cfg.RestateClient); err != nil {
			return fmt.Errorf("failed to bootstrap cert for %s: %w", domain, err)
		}
	}

	return nil
}

func (s *Service) ensureInfraDomain(ctx context.Context, domain string, restateClient hydrav1.CertificateServiceIngressClient) error {
	// Check if domain already has a cert via JOIN
	existingDomain, err := db.Query.FindCustomDomainWithCertByDomain(ctx, s.db.RO(), domain)
	if err != nil && !db.IsNotFound(err) {
		return fmt.Errorf("failed to check for existing domain: %w", err)
	}

	// If cert already exists, we're done
	if existingDomain.CertificateID.Valid && existingDomain.CertificateID.String != "" {
		s.logger.Info("infrastructure cert already exists", "domain", domain, "cert_id", existingDomain.CertificateID.String)
		return nil
	}

	// Create domain + challenge records if they don't exist
	if existingDomain.ID == "" {
		domainID := uid.New(uid.DomainPrefix)
		now := time.Now().UnixMilli()

		err = db.Query.UpsertCustomDomain(ctx, s.db.RW(), db.UpsertCustomDomainParams{
			ID:            domainID,
			WorkspaceID:   InfraWorkspaceID,
			Domain:        domain,
			ChallengeType: db.CustomDomainsChallengeTypeDNS01,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to create custom domain record: %w", err)
		}

		err = db.Query.InsertAcmeChallenge(ctx, s.db.RW(), db.InsertAcmeChallengeParams{
			WorkspaceID:   InfraWorkspaceID,
			DomainID:      domainID,
			Token:         "",
			Authorization: "",
			Status:        db.AcmeChallengesStatusWaiting,
			ChallengeType: db.AcmeChallengesChallengeTypeDNS01,
			CreatedAt:     now,
			UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
			ExpiresAt:     0,
		})
		if err != nil {
			return fmt.Errorf("failed to create ACME challenge record: %w", err)
		}

		s.logger.Info("created infrastructure domain records", "domain", domain, "domain_id", domainID)
	}

	// Trigger the ProcessChallenge workflow via Restate
	s.logger.Info("triggering certificate challenge workflow", "domain", domain)

	_, err = restateClient.ProcessChallenge().Send(ctx, &hydrav1.ProcessChallengeRequest{
		WorkspaceId: InfraWorkspaceID,
		Domain:      domain,
	})
	if err != nil {
		s.logger.Warn("failed to trigger certificate workflow, renewal cron will retry",
			"domain", domain,
			"error", err,
		)
	}

	return nil
}
