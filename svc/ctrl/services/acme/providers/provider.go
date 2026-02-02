package providers

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// ErrDomainNotFound is returned when a domain is not found in the database.
var ErrDomainNotFound = errors.New("domain not found")

var _ challenge.Provider = (*Provider)(nil)
var _ challenge.ProviderTimeout = (*Provider)(nil)

// DNSProvider is the interface for underlying DNS operations.
// Lego's route53.DNSProvider implements this.
type DNSProvider interface {
	Present(domain, token, keyAuth string) error
	CleanUp(domain, token, keyAuth string) error
	Timeout() (timeout, interval time.Duration)
}

// Provider wraps a DNS provider with database tracking and caching.
// It implements the lego challenge.Provider interface.
type Provider struct {
	db     db.Database
	logger logging.Logger
	dns    DNSProvider
	cache  cache.Cache[string, db.CustomDomain]
}

type ProviderConfig struct {
	DB          db.Database
	Logger      logging.Logger
	DNS         DNSProvider
	DomainCache cache.Cache[string, db.CustomDomain]
}

// NewProvider creates a new Provider that wraps a DNS provider with database tracking.
func NewProvider(cfg ProviderConfig) (*Provider, error) {
	err := assert.All(
		assert.NotNilAndNotZero(cfg.DB, "db is required"),
		assert.NotNilAndNotZero(cfg.Logger, "logger is required"),
		assert.NotNilAndNotZero(cfg.DNS, "dns provider is required"),
		assert.NotNilAndNotZero(cfg.DomainCache, "domain cache is required"),
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		db:     cfg.DB,
		logger: cfg.Logger,
		dns:    cfg.DNS,
		cache:  cfg.DomainCache,
	}, nil
}

// resolveDomain finds the best matching custom domain for a given domain.
// It queries for both the exact domain and wildcard (*.domain) in a single query,
// preferring exact matches.
func (p *Provider) resolveDomain(ctx context.Context, domain string) (db.CustomDomain, error) {
	wildcardDomain := "*." + domain
	cacheKey := domain + "|" + wildcardDomain

	dom, hit, err := p.cache.SWR(ctx, cacheKey,
		func(ctx context.Context) (db.CustomDomain, error) {
			return db.Query.FindCustomDomainByDomainOrWildcard(ctx, p.db.RO(), db.FindCustomDomainByDomainOrWildcardParams{
				Domain:   domain,
				Domain_2: wildcardDomain,
				Domain_3: domain,
			})
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

// Present creates a DNS TXT record for the ACME challenge and tracks it in the database.
func (p *Provider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()

	// Find domain - tries exact match first, then wildcard (*.domain)
	dom, err := p.resolveDomain(ctx, domain)
	if err != nil {
		return fmt.Errorf("failed to find domain %s: %w", domain, err)
	}

	p.logger.Info("presenting dns challenge", "domain", domain, "matched", dom.Domain)

	err = p.dns.Present(domain, token, keyAuth)
	if err != nil {
		return fmt.Errorf("failed to present DNS challenge for domain %s: %w", domain, err)
	}

	err = db.Query.UpdateAcmeChallengePending(ctx, p.db.RW(), db.UpdateAcmeChallengePendingParams{
		DomainID:      dom.ID,
		Status:        db.AcmeChallengesStatusPending,
		Token:         token,
		Authorization: keyAuth,
		UpdatedAt:     sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to store challenge for domain %s: %w", domain, err)
	}

	p.logger.Info("dns challenge presented successfully", "domain", domain)
	return nil
}

// CleanUp removes the DNS TXT record.
func (p *Provider) CleanUp(domain, token, keyAuth string) error {
	p.logger.Info("cleaning up dns challenge", "domain", domain)

	err := p.dns.CleanUp(domain, token, keyAuth)
	if err != nil {
		p.logger.Warn("failed to clean up dns challenge record", "error", err, "domain", domain)
	}

	return nil
}

// Timeout returns the timeout and polling interval for the DNS challenge.
func (p *Provider) Timeout() (timeout, interval time.Duration) {
	return p.dns.Timeout()
}
