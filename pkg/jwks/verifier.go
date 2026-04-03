package jwks

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/jwt"
)

// Verifier validates JWTs using keys resolved from a KeySet. It supports
// RS256 tokens where the signing key is identified by the kid header.
//
// The type parameter T should embed jwt.RegisteredClaims for standard JWT fields.
type Verifier[T any] struct {
	keySet   KeySet
	clock    clock.Clock
	issuer   string
	audience string
}

// VerifierOption configures a Verifier.
type VerifierOption func(*verifierConfig)

type verifierConfig struct {
	clock    clock.Clock
	issuer   string
	audience string
}

// WithIssuer requires the token's iss claim to match the given value.
func WithIssuer(iss string) VerifierOption {
	return func(c *verifierConfig) {
		c.issuer = iss
	}
}

// WithAudience requires the token's aud claim to include the given value.
func WithAudience(aud string) VerifierOption {
	return func(c *verifierConfig) {
		c.audience = aud
	}
}

// WithClock sets a custom clock for temporal claims validation.
func WithClock(clk clock.Clock) VerifierOption {
	return func(c *verifierConfig) {
		c.clock = clk
	}
}

// NewVerifier creates a Verifier that resolves signing keys from the given KeySet.
func NewVerifier[T any](keySet KeySet, opts ...VerifierOption) *Verifier[T] {
	cfg := verifierConfig{
		clock:    clock.New(),
		issuer:   "",
		audience: "",
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &Verifier[T]{
		keySet:   keySet,
		clock:    cfg.clock,
		issuer:   cfg.issuer,
		audience: cfg.audience,
	}
}

// jwtHeader is the decoded JWT header.
type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	Typ string `json:"typ"`
}

// Verify validates a JWT and returns the typed claims.
//
// Verification steps:
//  1. Parse and validate token structure (three dot-separated parts)
//  2. Decode header and extract kid + alg
//  3. Require alg=RS256
//  4. Resolve public key from KeySet using kid
//  5. Verify RSA-PKCS1v15-SHA256 signature
//  6. Decode and unmarshal claims
//  7. Validate registered claims (exp, nbf, iss, aud)
func (v *Verifier[T]) Verify(ctx context.Context, token string) (T, error) {
	var zero T

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return zero, fmt.Errorf("token must have 3 parts, got %d", len(parts))
	}

	// Decode header.
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return zero, fmt.Errorf("invalid header encoding: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return zero, fmt.Errorf("invalid header JSON: %w", err)
	}

	if header.Alg != "RS256" {
		return zero, fmt.Errorf("unsupported algorithm %q, expected RS256", header.Alg)
	}

	if header.Kid == "" {
		return zero, fmt.Errorf("missing kid in token header")
	}

	// Resolve the signing key.
	publicKey, err := v.keySet.GetKey(ctx, header.Kid)
	if err != nil {
		return zero, fmt.Errorf("resolving signing key: %w", err)
	}

	// Verify signature.
	signatureInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return zero, fmt.Errorf("invalid signature encoding: %w", err)
	}

	hash := sha256.Sum256([]byte(signatureInput))
	if err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return zero, fmt.Errorf("invalid signature: %w", err)
	}

	// Decode claims.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return zero, fmt.Errorf("invalid payload encoding: %w", err)
	}

	var claims T
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return zero, fmt.Errorf("invalid payload JSON: %w", err)
	}

	// Validate registered claims.
	var registered jwt.RegisteredClaims
	if err := json.Unmarshal(payloadJSON, &registered); err != nil {
		return zero, fmt.Errorf("invalid registered claims: %w", err)
	}

	now := v.clock.Now()
	if err := v.validateClaims(registered, now); err != nil {
		return zero, err
	}

	return claims, nil
}

// validateClaims checks temporal and identity claims.
func (v *Verifier[T]) validateClaims(c jwt.RegisteredClaims, now time.Time) error {
	if c.ExpiresAt != 0 && now.Unix() > c.ExpiresAt {
		return jwt.ErrTokenExpired
	}
	if c.NotBefore != 0 && now.Unix() < c.NotBefore {
		return jwt.ErrTokenNotYetValid
	}
	if v.issuer != "" && c.Issuer != v.issuer {
		return jwt.ErrInvalidIssuer
	}
	if v.audience != "" {
		found := slices.Contains(c.Audience, v.audience)
		if !found {
			return jwt.ErrInvalidAudience
		}
	}
	return nil
}
