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

// HTTPProvider implements the lego challenge.Provider interface for HTTP-01 challenges
// It stores challenges in the database where the gateway can retrieve them
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

// Present stores the challenge token in the database for the gateway to serve
// The gateway will intercept requests to /.well-known/acme-challenge/{token}
// and respond with the keyAuth value
func (p *HTTPProvider) Present(domain, token, keyAuth string) error {
	ctx := context.Background()
	dom, err := db.Query.FindDomainByDomain(ctx, p.db.RO(), domain)
	if err != nil {
		p.logger.Error("failed to find domain", "error", err, "domain", domain)
		return fmt.Errorf("failed to find domain: %w", err)
	}

	// Update the existing challenge record with the token and authorization
	err = db.Query.UpdateDomainChallengePending(ctx, p.db.RW(), db.UpdateDomainChallengePendingParams{
		DomainID:      dom.ID,
		Status:        db.DomainChallengesStatusPending,
		Token:         sql.NullString{String: token, Valid: true},
		Authorization: sql.NullString{String: keyAuth, Valid: true},
		UpdatedAt:     sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})

	if err != nil {
		p.logger.Error("failed to store challenge", "error", err, "domain", domain, "token", token)
		return fmt.Errorf("failed to store challenge: %w", err)
	}

	// Give the database time to replicate before Let's Encrypt tries to validate
	time.Sleep(2 * time.Second)

	return nil
}

// CleanUp removes the challenge from the database after validation
func (p *HTTPProvider) CleanUp(domain, token, keyAuth string) error {
	ctx := context.Background()

	dom, err := db.Query.FindDomainByDomain(ctx, p.db.RO(), domain)
	if err != nil {
		p.logger.Error("failed to find domain", "error", err, "domain", domain)
		return fmt.Errorf("failed to find domain: %w", err)
	}

	// Update the challenge status to mark it as verified
	err = db.Query.UpdateDomainChallengeStatus(ctx, p.db.RW(), db.UpdateDomainChallengeStatusParams{
		DomainID:  dom.ID,
		Status:    db.DomainChallengesStatusVerified,
		UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})

	if err != nil {
		p.logger.Warn("failed to clean up challenge", "error", err, "domain", domain, "token", token)
	}

	return nil
}
