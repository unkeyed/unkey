package jwt

import "time"

// Claims defines the interface for JWT claims that can be validated.
//
// Any type used with [Signer] or [Verifier] must implement this interface.
// The simplest approach is to embed [RegisteredClaims], which provides a
// Validate method that checks exp and nbf claims. Go's method promotion
// automatically satisfies this interface:
//
//	type MyClaims struct {
//	    jwt.RegisteredClaims
//	    TenantID string `json:"tenant_id"`
//	}
//	// MyClaims implements Claims via promoted RegisteredClaims.Validate
//
// Override Validate only if you need additional validation:
//
//	func (c MyClaims) Validate(now time.Time) error {
//	    if err := c.RegisteredClaims.Validate(now); err != nil {
//	        return err
//	    }
//	    if c.TenantID == "" {
//	        return errors.New("tenant_id is required")
//	    }
//	    return nil
//	}
type Claims interface {
	// Validate checks whether the claims are valid at the given time.
	//
	// Implementations should check temporal claims (exp, nbf) against the provided
	// time and return [ErrTokenExpired] or [ErrTokenNotYetValid] as appropriate.
	// Custom validation failures should return a descriptive error.
	//
	// The time parameter comes from the verifier's clock or an explicit override
	// passed to [Verifier.Verify]. It is not necessarily time.Now().
	Validate(now time.Time) error
}

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

// Validate checks the registered claims against the given time.
//
// Returns [ErrTokenExpired] if exp is set and the token has expired (now > exp).
// Returns [ErrTokenNotYetValid] if nbf is set and the token is not yet valid (now < nbf).
// Returns nil if all temporal claims are valid or not set.
//
// Zero values for exp and nbf are treated as "not set" and skip validation.
// Negative timestamps are compared normally; a negative exp in the past will
// result in [ErrTokenExpired].
//
// This method does not validate iss, sub, aud, iat, or jti. Applications requiring
// those checks must implement them separately after [Verifier.Verify] returns.
func (c RegisteredClaims) Validate(now time.Time) error {
	if c.ExpiresAt != 0 && now.Unix() > c.ExpiresAt {
		return ErrTokenExpired
	}
	if c.NotBefore != 0 && now.Unix() < c.NotBefore {
		return ErrTokenNotYetValid
	}
	return nil
}
