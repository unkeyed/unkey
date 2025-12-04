package providers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var ErrDomainNotFound = errors.New("domain not found")

var _ challenge.Provider = (*CloudflareProvider)(nil)
var _ challenge.ProviderTimeout = (*CloudflareProvider)(nil)

// CloudflareProvider implements the lego challenge.Provider interface for DNS-01 challenges.
// It uses Cloudflare DNS to store challenges and tracks them in the database.
type CloudflareProvider struct {
	db            db.Database
	logger        logging.Logger
	provider      *cloudflare.DNSProvider
	defaultDomain string
	cache         cache.Cache[string, db.CustomDomain]
}

type CloudflareProviderConfig struct {
	DB            db.Database
	Logger        logging.Logger
	APIToken      string                               // Cloudflare API token with Zone:Read, DNS:Edit permissions
	DefaultDomain string                               // Default domain for wildcard certificate handling
	DomainCache   cache.Cache[string, db.CustomDomain] // Optional: shared cache for domain lookups
}

// NewCloudflareProvider creates a new DNS-01 challenge provider using Cloudflare.
func NewCloudflareProvider(cfg CloudflareProviderConfig) (*CloudflareProvider, error) {
	config := cloudflare.NewDefaultConfig()
	config.PropagationTimeout = time.Minute * 5
	config.AuthToken = cfg.APIToken
	config.TTL = 60 * 10
	provider, err := cloudflare.NewDNSProviderConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare DNS provider: %w", err)
	}

	return &CloudflareProvider{
		db:            cfg.DB,
		logger:        cfg.Logger,
		provider:      provider,
		defaultDomain: cfg.DefaultDomain,
		cache:         cfg.DomainCache,
	}, nil
}

// Present creates a DNS TXT record for the ACME challenge.
func (p *CloudflareProvider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()

	searchDomain := domain
	if domain == p.defaultDomain {
		searchDomain = "*." + domain
	}

	dom, err := p.getDomain(ctx, searchDomain)
	if err != nil {
		return fmt.Errorf("failed to find domain %s: %w", searchDomain, err)
	}

	p.logger.Info("presenting dns challenge", "domain", domain, "token", "[REDACTED]")

	// Create the DNS challenge record using Cloudflare
	err = p.provider.Present(domain, token, keyAuth)
	if err != nil {
		return fmt.Errorf("failed to present DNS challenge for domain %s: %w", domain, err)
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
		// Don't cleanup DNS record - Let's Encrypt still needs it for validation
		// The DNS record will be cleaned up later in CleanUp() regardless of success/failure
		return fmt.Errorf("failed to store challenge for domain %s: %w", domain, err)
	}

	p.logger.Info("dns challenge presented successfully", "domain", domain)

	return nil
}

// CleanUp removes the DNS TXT record and updates the database
func (p *CloudflareProvider) CleanUp(domain, token, keyAuth string) error {
	p.logger.Info("cleaning up dns challenge", "domain", domain)

	// Clean up the DNS record first
	err := p.provider.CleanUp(domain, token, keyAuth)
	if err != nil {
		p.logger.Warn("failed to clean up dns challenge record", "error", err, "domain", domain)
	}

	return nil
}

// Timeout returns the timeout and polling interval for the DNS challenge.
func (p *CloudflareProvider) Timeout() (timeout, interval time.Duration) {
	return p.provider.Timeout()
}

func (p *CloudflareProvider) getDomain(ctx context.Context, domain string) (db.CustomDomain, error) {
	if p.cache != nil {
		dom, hit, err := p.cache.SWR(ctx, domain,
			func(ctx context.Context) (db.CustomDomain, error) {
				return db.Query.FindCustomDomainByDomain(ctx, p.db.RO(), domain)
			},
			caches.DefaultFindFirstOp,
		)
		if err != nil {
			return db.CustomDomain{}, err
		}
		if hit == cache.Null {
			return db.CustomDomain{}, ErrDomainNotFound
		}
		return dom, nil
	}
	return db.Query.FindCustomDomainByDomain(ctx, p.db.RO(), domain)
}
