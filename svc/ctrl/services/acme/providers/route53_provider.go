package providers

import (
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/route53"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
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
// Important: LEGO_DISABLE_CNAME_SUPPORT must be set to "true" before calling this
// function to prevent lego from following wildcard CNAMEs and failing zone lookup.
// This should be done once at application startup (see run.go).
//
// HostedZoneID is optional - if empty, lego will auto-discover the hosted zone
// based on the domain. Provide it only if auto-discovery fails (e.g., due to CNAMEs).
func NewRoute53Provider(cfg Route53Config) (*Provider, error) {
	config := route53.NewDefaultConfig()
	config.PropagationTimeout = time.Minute * 5
	config.TTL = 60 * 10 // 10 minutes
	config.AccessKeyID = cfg.AccessKeyID
	config.SecretAccessKey = cfg.SecretAccessKey
	config.Region = cfg.Region
	config.WaitForRecordSetsChanged = true

	// Only set HostedZoneID if provided - otherwise let lego auto-discover
	if cfg.HostedZoneID != "" {
		config.HostedZoneID = cfg.HostedZoneID
	}

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
