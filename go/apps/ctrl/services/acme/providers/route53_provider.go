package providers

import (
	"fmt"
	"os"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Route53Config struct {
	DB              db.Database
	Logger          logging.Logger
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	DomainCache     cache.Cache[string, db.CustomDomain]
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
func NewRoute53Provider(cfg Route53Config) (*Provider, error) {
	// Disable CNAME following in lego to prevent it from following wildcard CNAMEs
	// and failing to find the correct Route53 zone.
	os.Setenv("LEGO_DISABLE_CNAME_SUPPORT", "true")

	config := route53.NewDefaultConfig()
	config.PropagationTimeout = time.Minute * 5
	config.TTL = 60 * 10 // 10 minutes
	config.AccessKeyID = cfg.AccessKeyID
	config.SecretAccessKey = cfg.SecretAccessKey
	config.Region = cfg.Region
	config.HostedZoneID = cfg.HostedZoneID
	config.WaitForRecordSetsChanged = true

	dns, err := route53.NewDNSProviderConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Route53 DNS provider: %w", err)
	}

	return NewProvider(ProviderConfig{
		DB:          cfg.DB,
		Logger:      cfg.Logger,
		DNS:         dns,
		DomainCache: cfg.DomainCache,
	})
}
