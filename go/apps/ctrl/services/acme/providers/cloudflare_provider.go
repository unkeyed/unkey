package providers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ challenge.Provider = (*CloudflareProvider)(nil)

// CloudflareProvider implements the lego challenge.Provider interface for DNS-01 challenges
// It uses Cloudflare DNS to store challenges and tracks them in the database
type CloudflareProvider struct {
	db            db.Database
	logger        logging.Logger
	provider      *cloudflare.DNSProvider
	defaultDomain string
}

type CloudflareProviderConfig struct {
	DB            db.Database
	Logger        logging.Logger
	APIToken      string // Cloudflare API token with Zone:Read, DNS:Edit permissions
	DefaultDomain string // Default domain for wildcard certificate handling
}

// NewCloudflareProvider creates a new DNS-01 challenge provider using Cloudflare
func NewCloudflareProvider(cfg CloudflareProviderConfig) (*CloudflareProvider, error) {
	config := cloudflare.NewDefaultConfig()
	config.AuthToken = cfg.APIToken
	config.TTL = 120 // 2 minutes TTL for challenge records

	provider, err := cloudflare.NewDNSProviderConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare DNS provider: %w", err)
	}

	return &CloudflareProvider{
		db:            cfg.DB,
		logger:        cfg.Logger,
		provider:      provider,
		defaultDomain: cfg.DefaultDomain,
	}, nil
}

// Present creates a DNS TXT record for the ACME challenge
func (p *CloudflareProvider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()

	// Find domain in database to track the challenge
	// For DNS-01 challenges on the default domain, Let's Encrypt passes the base domain
	// but we store the wildcard domain in the database
	searchDomain := domain
	if domain == p.defaultDomain {
		// This is our default domain - look for the wildcard version
		searchDomain = "*." + domain
	}

	dom, err := db.Query.FindDomainByDomain(ctx, p.db.RO(), searchDomain)
	if err != nil {
		p.logger.Error("failed to find domain", "error", err, "domain", searchDomain)
		return fmt.Errorf("failed to find domain: %w", err)
	}

	p.logger.Info("presenting DNS challenge", "domain", domain, "token", token)

	// Create the DNS challenge record using Cloudflare
	err = p.provider.Present(domain, token, keyAuth)
	if err != nil {
		p.logger.Error("failed to present DNS challenge", "error", err, "domain", domain, "token", token)
		return fmt.Errorf("failed to present DNS challenge: %w", err)
	}

	// Update the database to track the challenge
	err = db.Query.UpdateAcmeChallengePending(ctx, p.db.RW(), db.UpdateAcmeChallengePendingParams{
		DomainID:      dom.ID,
		Status:        db.AcmeChallengesStatusPending,
		Token:         token,
		Authorization: keyAuth,
		UpdatedAt:     sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})

	if err != nil {
		p.logger.Error("failed to store challenge in database", "error", err, "domain", domain, "token", token)
		// Don't cleanup DNS record - Let's Encrypt still needs it for validation
		// The DNS record will be cleaned up later in CleanUp() regardless of success/failure
		return fmt.Errorf("failed to store challenge: %w", err)
	}

	p.logger.Info("DNS challenge presented successfully", "domain", domain, "token", token)

	// Give DNS time to propagate before Let's Encrypt validates
	time.Sleep(30 * time.Second)

	return nil
}

// CleanUp removes the DNS TXT record and updates the database
func (p *CloudflareProvider) CleanUp(domain, token, keyAuth string) error {
	p.logger.Info("cleaning up DNS challenge", "domain", domain, "token", token)

	// Clean up the DNS record first
	err := p.provider.CleanUp(domain, token, keyAuth)
	if err != nil {
		p.logger.Warn("failed to clean up DNS challenge record", "error", err, "domain", domain, "token", token)
	}

	return nil
}

// Timeout returns the timeout and polling interval for the DNS challenge
func (p *CloudflareProvider) Timeout() (timeout, interval time.Duration) {
	return p.provider.Timeout()
}
