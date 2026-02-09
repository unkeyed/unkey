package providers

import (
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

var _ challenge.Provider = (*HTTPProvider)(nil)
var _ challenge.ProviderTimeout = (*HTTPProvider)(nil)

// httpDNS implements the DNSProvider interface for HTTP-01 challenges.
// It stores challenges in the database where the gateway can retrieve them.
type httpDNS struct {
	db db.Database
}

// Present stores the challenge token in the database for the gateway to serve.
// The gateway will intercept requests to /.well-known/acme-challenge/{token}
// and respond with the keyAuth value.
func (h *httpDNS) Present(domain, token, keyAuth string) error {
	logger.Info("presenting http-01 challenge", "domain", domain)
	// The actual DB update is handled by the generic Provider wrapper
	return nil
}

// CleanUp is a no-op for HTTP-01 - the token remains in DB until overwritten
func (h *httpDNS) CleanUp(domain, token, keyAuth string) error {
	logger.Info("cleaning up http-01 challenge", "domain", domain)
	return nil
}

// Timeout returns custom timeout and check interval for HTTP-01 challenges.
// HTTP challenges typically resolve faster than DNS.
func (h *httpDNS) Timeout() (time.Duration, time.Duration) {
	return 90 * time.Second, 3 * time.Second
}

// HTTPProvider wraps httpDNS with the generic Provider for DB tracking and caching.
// This is a type alias to make it clear this is an HTTP-01 provider.
type HTTPProvider = Provider

type HTTPConfig struct {
	DB          db.Database
	DomainCache cache.Cache[string, db.CustomDomain]
}

// NewHTTPProvider creates a new HTTP-01 challenge provider.
func NewHTTPProvider(cfg HTTPConfig) (*HTTPProvider, error) {
	return NewProvider(ProviderConfig{
		DB: cfg.DB,
		DNS: &httpDNS{
			db: cfg.DB,
		},
		DomainCache: cfg.DomainCache,
	})
}
