package certificate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restateIngress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

// InfraWorkspaceID is the workspace ID for infrastructure certificates. Infrastructure
// certs are owned by this synthetic workspace rather than a customer workspace, allowing
// them to be managed separately and avoiding conflicts with customer domain records.
const InfraWorkspaceID = "unkey_internal"

// BootstrapConfig holds configuration for [Service.BootstrapInfraCerts].
type BootstrapConfig struct {
	// DefaultDomain is the base domain for customer deployments. If set, a wildcard
	// certificate for "*.{DefaultDomain}" is provisioned to terminate TLS for all
	// customer subdomains.
	DefaultDomain string

	// RegionalDomain is the base domain for cross-region communication between
	// frontline instances. Combined with each entry in Regions to create per-region
	// wildcard certificates.
	RegionalDomain string

	// Regions lists all deployment regions. For each region, a wildcard certificate
	// is created as "*.{region}.{RegionalDomain}" to secure inter-region traffic.
	Regions []string

	// Restate is the ingress client used to trigger [Service.ProcessChallenge] workflows.
	// Infrastructure cert bootstrapping delegates to the standard challenge flow rather
	// than implementing separate certificate logic.
	Restate *restateIngress.Client
}

// BootstrapInfraCerts provisions wildcard certificates for platform infrastructure.
//
// This method ensures the platform has valid TLS certificates for its own domains
// before serving customer traffic. It creates database records for each infrastructure
// domain and triggers [Service.ProcessChallenge] via Restate to obtain certificates.
//
// The method is idempotent: domains with existing valid certificates are skipped,
// and domains with pending challenges are not re-triggered. This makes it safe to
// call on every service startup without risking duplicate certificate requests.
//
// Returns nil without error if DNSProvider is not configured, since infrastructure
// certs require DNS-01 challenges for wildcards. Logs a warning in this case.
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
	if cfg.RegionalDomain != "" {
		for _, region := range cfg.Regions {
			domains = append(domains, fmt.Sprintf("*.%s.%s", region, cfg.RegionalDomain))
		}
	}

	if len(domains) == 0 {
		s.logger.Info("no infrastructure domains configured, skipping cert bootstrap")
		return nil
	}

	for _, domain := range domains {
		if err := s.ensureInfraDomain(ctx, domain, cfg.Restate); err != nil {
			return fmt.Errorf("failed to bootstrap cert for %s: %w", domain, err)
		}
	}

	return nil
}

func (s *Service) ensureInfraDomain(ctx context.Context, domain string, restate *restateIngress.Client) error {
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
			ID:                 domainID,
			WorkspaceID:        InfraWorkspaceID,
			ProjectID:          InfraWorkspaceID,
			EnvironmentID:      InfraWorkspaceID,
			Domain:             domain,
			ChallengeType:      db.CustomDomainsChallengeTypeDNS01,
			VerificationStatus: db.CustomDomainsVerificationStatusVerified, // Pre-verified for infra domains
			VerificationToken:  "",                                         // Not needed for infra domains (already verified)
			TargetCname:        uid.DNS1035(16),                            // Unique target (not used for DNS-01 but required for uniqueness)
			CreatedAt:          now,
			UpdatedAt:          sql.NullInt64{Int64: now, Valid: true},
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
	// Use domain as key so multiple domains can be processed in parallel
	s.logger.Info("triggering certificate challenge workflow", "domain", domain)

	certClient := hydrav1.NewCertificateServiceIngressClient(restate, domain)
	_, err = certClient.ProcessChallenge().Send(ctx, &hydrav1.ProcessChallengeRequest{
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
