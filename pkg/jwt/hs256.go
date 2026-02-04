package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
)

// HS256Signer creates signed JSON Web Tokens using the HS256 algorithm
// (HMAC with SHA-256).
//
// HS256 is a symmetric algorithm: the same secret is used for both signing and
// verification. Use this when the signing and verifying parties can securely
// share a secret, such as a single service signing tokens for its own use.
//
// For scenarios where the signing party should not be able to verify (or vice versa),
// use [RS256Signer] with asymmetric keys instead.
//
// The type parameter T must embed [RegisteredClaims] to include standard JWT fields.
// HS256Signer is safe for concurrent use; the secret is captured at construction.
type HS256Signer[T any] struct {
	secret []byte
}

// Ensure HS256Signer implements Signer interface.
var _ Signer[RegisteredClaims] = (*HS256Signer[RegisteredClaims])(nil)

// NewHS256Signer creates an HS256Signer with the given secret key.
//
// The secret should be a cryptographically random value. While any non-empty
// secret is accepted, secrets shorter than 32 bytes (256 bits) provide less
// security than the underlying SHA-256 hash and are not recommended for
// production use.
//
// The secret slice is stored directly; the caller should not modify it after
// passing it to this function.
//
// Returns an error if the secret is nil or empty.
func NewHS256Signer[T any](secret []byte) (*HS256Signer[T], error) {
	if err := assert.NotEmpty(secret, "secret must not be empty"); err != nil {
		return nil, err
	}

	return &HS256Signer[T]{secret: secret}, nil
}

// Sign creates a signed JWT from the given claims.
//
// The token header is {"alg":"HS256","typ":"JWT"}. The claims are JSON-serialized
// and the signature is computed as HMAC-SHA256(secret, base64url(header) + "." + base64url(claims)).
//
// Returns the complete JWT in the format "header.payload.signature" where each
// segment uses base64url encoding without padding.
//
// Returns an error only if JSON marshaling fails, which indicates the claims
// contain types that cannot be serialized (channels, functions, etc.).
func (s *HS256Signer[T]) Sign(claims T) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT header: %w", err)
	}

	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signatureInput := encodedHeader + "." + encodedPayload

	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(signatureInput))
	signature := mac.Sum(nil)

	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	return signatureInput + "." + encodedSignature, nil
}

// HS256Verifier validates JSON Web Tokens signed with the HS256 algorithm
// (HMAC with SHA-256).
//
// The verifier only accepts tokens with alg=HS256 in the header. Tokens with
// any other algorithm (including "none", RS256, etc.) are rejected. This
// prevents algorithm confusion attacks where an attacker might try to use
// a different algorithm to forge a valid signature.
//
// Signature comparison uses [crypto/hmac.Equal] which is constant-time to
// prevent timing attacks.
//
// The type parameter T must embed [RegisteredClaims] to include standard JWT fields.
// HS256Verifier is safe for concurrent use.
type HS256Verifier[T any] struct {
	secret []byte
	clock  clock.Clock
	config verifyConfig
}

// Ensure HS256Verifier implements Verifier interface.
var _ Verifier[RegisteredClaims] = (*HS256Verifier[RegisteredClaims])(nil)

// NewHS256Verifier creates an HS256Verifier with the given secret key.
//
// The secret must match the secret used by the corresponding [HS256Signer].
// Tokens signed with a different secret will fail verification.
//
// Options can be used to configure verification behavior:
//
//	verifier, err := NewHS256Verifier[MyClaims](secret,
//	    jwt.WithIssuer("https://auth.example.com"),
//	    jwt.WithAudience("my-service"),
//	)
//
// Returns an error if the secret is nil or empty.
func NewHS256Verifier[T any](secret []byte, opts ...VerifyOption) (*HS256Verifier[T], error) {
	if err := assert.NotEmpty(secret, "secret must not be empty"); err != nil {
		return nil, err
	}

	cfg := verifyConfig{clock: clock.New(), issuer: "", audience: ""}
	for _, opt := range opts {
		opt(&cfg)
	}

	return &HS256Verifier[T]{secret: secret, clock: cfg.clock, config: cfg}, nil
}

// Verify validates a JWT and returns the typed claims.
//
// Verification proceeds in order:
//  1. Structure: token must have exactly three dot-separated parts
//  2. Header decoding: first part must be valid base64url JSON with alg field
//  3. Algorithm check: alg must be exactly "HS256"
//  4. Signature verification: HMAC-SHA256 of header.payload must match
//  5. Payload decoding: second part must be valid base64url JSON
//  6. Claims validation: [Claims.Validate] is called with the verification time
//
// The optional time parameter overrides the clock for claims validation. This is
// useful for testing tokens at specific times. In production, omit this parameter.
// If multiple times are passed, only the first is used.
//
// Returns [ErrTokenExpired] if the token's exp claim is in the past, and
// [ErrTokenNotYetValid] if the nbf claim is in the future. Other errors indicate
// malformed tokens, wrong algorithms, or invalid signatures.
//
// On any error, the returned claims value is the zero value of T. Do not use
// partial claims from failed verification.
func (v *HS256Verifier[T]) Verify(token string, at ...time.Time) (T, error) {
	verifyAt := v.clock.Now()
	if len(at) > 0 {
		verifyAt = at[0]
	}
	var claims T

	if err := assert.NotEmpty(token, "token must not be empty"); err != nil {
		return claims, err
	}

	parts := strings.Split(token, ".")
	if err := assert.Equal(len(parts), 3, "token must have 3 parts"); err != nil {
		return claims, err
	}

	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, fmt.Errorf("invalid header encoding: %w", err)
	}

	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return claims, fmt.Errorf("invalid header JSON: %w", err)
	}

	if err := assert.Equal(header.Alg, "HS256", "unsupported algorithm"); err != nil {
		return claims, err
	}

	signatureInput := parts[0] + "." + parts[1]
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, fmt.Errorf("invalid signature encoding: %w", err)
	}

	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(signatureInput))
	expectedSignature := mac.Sum(nil)

	if !hmac.Equal(providedSignature, expectedSignature) {
		return claims, fmt.Errorf("invalid signature")
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, fmt.Errorf("invalid payload encoding: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return claims, fmt.Errorf("invalid payload JSON: %w", err)
	}

	// Unmarshal registered claims separately for validation
	var registered RegisteredClaims
	if err := json.Unmarshal(payloadJSON, &registered); err != nil {
		return claims, fmt.Errorf("invalid registered claims: %w", err)
	}

	if err := registered.validate(verifyAt, &v.config); err != nil {
		return claims, err
	}

	return claims, nil
}
