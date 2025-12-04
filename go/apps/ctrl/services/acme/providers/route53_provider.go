package providers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ challenge.Provider = (*Route53Provider)(nil)
var _ challenge.ProviderTimeout = (*Route53Provider)(nil)

// Route53Provider implements the lego challenge.Provider interface for DNS-01 challenges
// using AWS Route53. It tracks challenges in the database.
type Route53Provider struct {
	db            db.Database
	logger        logging.Logger
	provider      *route53.DNSProvider
	defaultDomain string
	cache         cache.Cache[string, db.CustomDomain]
}

type Route53ProviderConfig struct {
	DB              db.Database
	Logger          logging.Logger
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	DefaultDomain   string
	DomainCache     cache.Cache[string, db.CustomDomain] // Optional: shared cache for domain lookups
	// HostedZoneID bypasses zone auto-discovery. Required when domains have CNAMEs
	// that would confuse the zone lookup (e.g., wildcard CNAMEs to load balancers).
	HostedZoneID string
}

// NewRoute53Provider creates a new DNS-01 challenge provider using AWS Route53.
//
// CNAME following is always disabled via LEGO_DISABLE_CNAME_SUPPORT to prevent lego
// from following wildcard CNAMEs (e.g., *.example.com -> loadbalancer.aws.com) and
// failing to find the correct Route53 zone.
//
// HostedZoneID should be provided to explicitly specify which Route53 zone to use,
// bypassing zone auto-discovery.
func NewRoute53Provider(cfg Route53ProviderConfig) (*Route53Provider, error) {
	// disable CNAME following in lego
	os.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")

	config := route53.NewDefaultConfig()
	config.PropagationTimeout = time.Minute * 5
	config.TTL = 60 * 10 // 10 minutes
	config.AccessKeyID = cfg.AccessKeyID
	config.SecretAccessKey = cfg.SecretAccessKey
	config.Region = cfg.Region
	config.HostedZoneID = cfg.HostedZoneID
	config.WaitForRecordSetsChanged = true

	provider, err := route53.NewDNSProviderConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Route53 DNS provider: %w", err)
	}

	return &Route53Provider{
		db:            cfg.DB,
		logger:        cfg.Logger,
		provider:      provider,
		defaultDomain: cfg.DefaultDomain,
		cache:         cfg.DomainCache,
	}, nil
}

// Present creates a DNS TXT record for the ACME challenge.
func (p *Route53Provider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()

	searchDomain := domain
	if domain == p.defaultDomain {
		searchDomain = "*." + domain
	}

	dom, err := p.getDomain(ctx, searchDomain)
	if err != nil {
		return fmt.Errorf("failed to find domain %s: %w", searchDomain, err)
	}

	p.logger.Info("presenting dns challenge via route53", "domain", domain)

	err = p.provider.Present(domain, token, keyAuth)
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

	p.logger.Info("dns challenge presented successfully via route53", "domain", domain)
	return nil
}

// CleanUp removes the DNS TXT record.
func (p *Route53Provider) CleanUp(domain, token, keyAuth string) error {
	p.logger.Info("cleaning up dns challenge via route53", "domain", domain)

	err := p.provider.CleanUp(domain, token, keyAuth)
	if err != nil {
		p.logger.Warn("failed to clean up dns challenge record", "error", err, "domain", domain)
	}

	return nil
}

// Timeout returns the timeout and polling interval for the DNS challenge.
func (p *Route53Provider) Timeout() (timeout, interval time.Duration) {
	return p.provider.Timeout()
}

func (p *Route53Provider) getDomain(ctx context.Context, domain string) (db.CustomDomain, error) {
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
