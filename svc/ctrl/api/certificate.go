package api

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/uid"
)

// certificateBootstrap handles ACME certificate bootstrapping and renewal.
// This ensures wildcard certificates exist for the default domain,
// regional apex domains, and starts the certificate renewal cron job.
type certificateBootstrap struct {
	logger         logging.Logger
	database       db.Database
	defaultDomain  string
	regionalDomain string
	regions        []string
	restateClient  hydrav1.CertificateServiceIngressClient
}

func (c *certificateBootstrap) run(ctx context.Context) {
	// Wait for services to be ready
	time.Sleep(5 * time.Second)

	// Bootstrap default domain wildcard (e.g., *.unkey.app)
	if c.defaultDomain != "" {
		c.bootstrapDomain(ctx, "*."+c.defaultDomain)
	}

	// Bootstrap per-region wildcards (e.g., *.us-west-2.aws.unkey.cloud)
	if c.regionalDomain != "" {
		for _, region := range c.regions {
			domain := fmt.Sprintf("*.%s.%s", region, c.regionalDomain)
			c.bootstrapDomain(ctx, domain)
		}
	}

	c.startCertRenewalCron(ctx)
}

func (c *certificateBootstrap) startCertRenewalCron(ctx context.Context) {
	_, err := c.restateClient.RenewExpiringCertificates().Send(
		ctx,
		&hydrav1.RenewExpiringCertificatesRequest{
			DaysBeforeExpiry: 30,
		},
		restate.WithIdempotencyKey("cert-renewal-cron-startup"),
	)
	if err != nil {
		c.logger.Warn("failed to start certificate renewal cron", "error", err)
		return
	}
	c.logger.Info("Certificate renewal cron job started")
}

// bootstrapDomain ensures a wildcard domain and ACME challenge exist for the given domain.
//
// This helper function creates the necessary database records for automatic
// wildcard certificate issuance during startup. It checks if the wildcard
// domain already exists and creates both the custom domain record and
// ACME challenge record if needed.
//
// The function uses "unkey_internal" as the workspace ID for
// platform-managed resources, ensuring separation from user workspaces.
func (c *certificateBootstrap) bootstrapDomain(ctx context.Context, domain string) {
	// Check if the domain already exists
	_, err := db.Query.FindCustomDomainByDomain(ctx, c.database.RO(), domain)
	if err == nil {
		c.logger.Info("Domain already exists", "domain", domain)
		return
	}
	if !db.IsNotFound(err) {
		c.logger.Error("Failed to check for existing domain", "error", err, "domain", domain)
		return
	}

	// Create the custom domain record
	domainID := uid.New(uid.DomainPrefix)
	now := time.Now().UnixMilli()

	// Use "unkey_internal" as the workspace/project/environment for platform-managed resources
	// Infrastructure wildcard domains are pre-verified (we control DNS via Route53)
	internalID := "unkey_internal"
	err = db.Query.UpsertCustomDomain(ctx, c.database.RW(), db.UpsertCustomDomainParams{
		ID:                 domainID,
		WorkspaceID:        internalID,
		ProjectID:          internalID,
		EnvironmentID:      internalID,
		Domain:             domain,
		ChallengeType:      db.CustomDomainsChallengeTypeDNS01,
		VerificationStatus: db.CustomDomainsVerificationStatusVerified, // Pre-verified for infra domains
		VerificationToken:  "",                                         // Not needed for infra domains (already verified)
		TargetCname:        uid.DNS1035(16),                            // Unique target (not used for DNS-01 but required for uniqueness)
		CreatedAt:          now,
		UpdatedAt:          sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		c.logger.Error("Failed to create domain", "error", err, "domain", domain)
		return
	}

	// Create the ACME challenge record with status 'waiting' so the renewal cron picks it up
	err = db.Query.InsertAcmeChallenge(ctx, c.database.RW(), db.InsertAcmeChallengeParams{
		WorkspaceID:   internalID,
		DomainID:      domainID,
		Token:         "",
		Authorization: "",
		Status:        db.AcmeChallengesStatusWaiting,
		ChallengeType: db.AcmeChallengesChallengeTypeDNS01,
		CreatedAt:     now,
		UpdatedAt:     sql.NullInt64{Int64: now, Valid: true},
		ExpiresAt:     0, // Will be set when certificate is issued
	})
	if err != nil {
		c.logger.Error("Failed to create ACME challenge for domain", "error", err, "domain", domain)
		return
	}

	c.logger.Info("Bootstrapped domain for certificate issuance", "domain", domain)
}
