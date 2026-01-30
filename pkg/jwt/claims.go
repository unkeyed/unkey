package jwt

import (
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
)

// Maximum sizes for registered claim values in bytes.
// These limits protect against denial of service attacks using oversized tokens.
const (
	// MaxIssuerSize is the maximum allowed length for the iss claim.
	MaxIssuerSize = 255

	// MaxSubjectSize is the maximum allowed length for the sub claim.
	MaxSubjectSize = 255

	// MaxAudienceEntrySize is the maximum allowed length for each aud entry.
	MaxAudienceEntrySize = 255

	// MaxAudienceCount is the maximum number of entries allowed in the aud claim.
	MaxAudienceCount = 10

	// MaxIDSize is the maximum allowed length for the jti claim.
	MaxIDSize = 255
)

// RegisteredClaims contains the standard JWT registered claim names as defined
// in RFC 7519 Section 4.1. Embed this struct in your custom claims type to
// automatically include the standard fields with correct JSON serialization.
//
// All fields are optional per the JWT specification. Zero values are treated as
// "not set" and are omitted from JSON output (via omitempty) and skipped during
// validation.
//
// Example:
//
//	type SessionClaims struct {
//	    jwt.RegisteredClaims
//	    SessionID string `json:"sid"`
//	}
//
//	claims := SessionClaims{
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        Subject:   "user-123",
//	        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
//	        IssuedAt:  time.Now().Unix(),
//	    },
//	    SessionID: "sess-abc",
//	}
type RegisteredClaims struct {
	// Issuer identifies the principal that issued the JWT. Typically a URL or
	// service name like "https://auth.example.com" or "auth-service".
	Issuer string `json:"iss,omitempty"`

	// Subject identifies the principal that is the subject of the JWT. This is
	// usually a user ID, API key ID, or other stable identifier for the entity
	// the token represents.
	Subject string `json:"sub,omitempty"`

	// Audience identifies the recipients that the JWT is intended for. Each entry
	// is typically a service name or URL. A token should only be accepted if the
	// verifying service is in this list.
	//
	// This field serializes as a JSON array. Tokens from libraries that serialize
	// single audiences as a bare string (without array brackets) will fail to parse.
	Audience []string `json:"aud,omitempty"`

	// ExpiresAt is the Unix timestamp (seconds since epoch) after which the JWT
	// must not be accepted for processing. The current implementation considers
	// a token expired when now > exp; at exactly exp, the token is still valid.
	ExpiresAt int64 `json:"exp,omitempty"`

	// NotBefore is the Unix timestamp before which the JWT must not be accepted.
	// The token becomes valid at exactly nbf (inclusive).
	NotBefore int64 `json:"nbf,omitempty"`

	// IssuedAt is the Unix timestamp at which the JWT was issued. This is
	// informational and not validated by [RegisteredClaims.Validate].
	IssuedAt int64 `json:"iat,omitempty"`

	// ID provides a unique identifier for the JWT. This can be used to prevent
	// token replay by tracking which token IDs have been used. Not validated
	// by this package; applications must implement their own tracking.
	ID string `json:"jti,omitempty"`
}

// validate checks the registered claims against the given time and verifier config.
//
// This is called internally by the verifier and validates:
//   - Size limits on string claims (iss, sub, jti, aud)
//   - Temporal claims (exp, nbf)
//   - Issuer matching (if configured)
//   - Audience matching (if configured)
func (c RegisteredClaims) validate(now time.Time, cfg *verifyConfig) error {
	if err := assert.All(
		assert.LessOrEqual(len(c.Issuer), MaxIssuerSize, "iss exceeds max size"),
		assert.LessOrEqual(len(c.Subject), MaxSubjectSize, "sub exceeds max size"),
		assert.LessOrEqual(len(c.ID), MaxIDSize, "jti exceeds max size"),
		assert.LessOrEqual(len(c.Audience), MaxAudienceCount, "aud exceeds max entries"),
	); err != nil {
		return err
	}
	for _, aud := range c.Audience {
		if err := assert.LessOrEqual(len(aud), MaxAudienceEntrySize, "aud entry exceeds max size"); err != nil {
			return err
		}
	}

	if c.ExpiresAt != 0 && now.Unix() > c.ExpiresAt {
		return ErrTokenExpired
	}
	if c.NotBefore != 0 && now.Unix() < c.NotBefore {
		return ErrTokenNotYetValid
	}

	if cfg.issuer != "" && c.Issuer != cfg.issuer {
		return ErrInvalidIssuer
	}
	if cfg.audience != "" && !slices.Contains(c.Audience, cfg.audience) {
		return ErrInvalidAudience
	}

	return nil
}

// verifyConfig holds configuration for token verification.
type verifyConfig struct {
	issuer   string
	audience string
	clock    clock.Clock
}

// VerifyOption configures token verification behavior.
type VerifyOption func(*verifyConfig)

// WithIssuer requires tokens to have the specified issuer (iss claim).
// If the token's issuer does not match, verification fails with [ErrInvalidIssuer].
func WithIssuer(iss string) VerifyOption {
	return func(cfg *verifyConfig) {
		cfg.issuer = iss
	}
}

// WithAudience requires tokens to include the specified audience in their aud claim.
// If the token's audience does not contain this value, verification fails with [ErrInvalidAudience].
func WithAudience(aud string) VerifyOption {
	return func(cfg *verifyConfig) {
		cfg.audience = aud
	}
}

// WithClock sets a custom clock for temporal claims validation.
// This is primarily useful for testing with fixed or controlled time.
func WithClock(c clock.Clock) VerifyOption {
	return func(cfg *verifyConfig) {
		cfg.clock = c
	}
}
