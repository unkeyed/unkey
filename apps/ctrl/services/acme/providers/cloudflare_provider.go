package providers

import (
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/providers/dns/cloudflare"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type CloudflareConfig struct {
	DB          db.Database
	Logger      logging.Logger
	APIToken    string // Cloudflare API token with Zone:Read, DNS:Edit permissions
	DomainCache cache.Cache[string, db.CustomDomain]
}

// NewCloudflareProvider creates a new DNS-01 challenge provider using Cloudflare.
func NewCloudflareProvider(cfg CloudflareConfig) (*Provider, error) {
	config := cloudflare.NewDefaultConfig()
	config.PropagationTimeout = time.Minute * 5
	config.AuthToken = cfg.APIToken
	config.TTL = 60 * 10

	dns, err := cloudflare.NewDNSProviderConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloudflare DNS provider: %w", err)
	}

	return NewProvider(ProviderConfig{
		DB:          cfg.DB,
		Logger:      cfg.Logger,
		DNS:         dns,
		DomainCache: cfg.DomainCache,
	})
}
