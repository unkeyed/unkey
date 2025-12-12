package providers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/challenge"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

var _ challenge.Provider = (*HTTPProvider)(nil)
var _ challenge.ProviderTimeout = (*HTTPProvider)(nil)

// HTTPProvider implements the lego challenge.Provider interface for HTTP-01 challenges
// It stores challenges in the database where the sentinel can retrieve them
type HTTPProvider struct {
	db     db.Database
	logger logging.Logger
}

type HTTPProviderConfig struct {
	DB     db.Database
	Logger logging.Logger
}

// NewHTTPProvider creates a new HTTP-01 challenge provider
func NewHTTPProvider(cfg HTTPProviderConfig) *HTTPProvider {
	return &HTTPProvider{
		db:     cfg.DB,
		logger: cfg.Logger,
	}
}

// Present stores the challenge token in the database for the sentinel to serve
// The sentinel will intercept requests to /.well-known/acme-challenge/{token}
// and respond with the keyAuth value
func (p *HTTPProvider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()
	dom, err := db.Query.FindCustomDomainByDomain(ctx, p.db.RO(), domain)
	if err != nil {
		return fmt.Errorf("failed to find domain %s: %w", domain, err)
	}

	// Update the existing challenge record with the token and authorization
	err = db.Query.UpdateAcmeChallengePending(ctx, p.db.RW(), db.UpdateAcmeChallengePendingParams{
		DomainID:      dom.ID,
		Status:        db.AcmeChallengesStatusPending,
		Token:         token,
		Authorization: keyAuth,
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})

	if err != nil {
		return fmt.Errorf("failed to store challenge for domain %s: %w", domain, err)
	}

	return nil
}

// CleanUp removes the challenge token from the database after validation
func (p *HTTPProvider) CleanUp(domain, token, keyAuth string) error {
	ctx := context.Background()

	dom, err := db.Query.FindCustomDomainByDomain(ctx, p.db.RO(), domain)
	if err != nil {
		return fmt.Errorf("failed to find domain %s during cleanup: %w", domain, err)
	}

	// Clear the token and authorization so the sentinel stops serving the challenge
	// Don't change the status - it should remain as set by the certificate workflow
	err = db.Query.ClearAcmeChallengeTokens(ctx, p.db.RW(), db.ClearAcmeChallengeTokensParams{
		Token:         "", // Clear token
		Authorization: "", // Clear authorization
		UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		DomainID:      dom.ID,
	})

	if err != nil {
		p.logger.Warn("failed to clean up challenge token", "error", err, "domain", domain)
	}

	return nil
}

// Timeout returns custom timeout and check interval for HTTP-01 challenges
// Returns (timeout, interval) - how long to wait and time between checks
func (p *HTTPProvider) Timeout() (time.Duration, time.Duration) {
	// HTTP challenges typically resolve faster than DNS, but give some buffer
	// 90 seconds timeout, 3 second check interval
	return 90 * time.Second, 3 * time.Second
}
