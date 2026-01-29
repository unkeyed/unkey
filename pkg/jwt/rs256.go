package jwt

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
)

// RS256Signer creates signed JSON Web Tokens using the RS256 algorithm
// (RSA PKCS#1 v1.5 with SHA-256).
//
// RS256 is an asymmetric algorithm: tokens are signed with a private key and
// verified with the corresponding public key. Use this for service-to-service
// authentication where the signing service should be the only entity capable
// of creating valid tokens, while multiple services can verify them.
//
// For simpler scenarios where the same service signs and verifies, consider
// [HS256Signer] which uses symmetric keys.
//
// The type parameter T specifies the claims type, which must implement [Claims].
// RS256Signer is safe for concurrent use; the private key is captured at construction.
type RS256Signer[T Claims] struct {
	privateKey *rsa.PrivateKey
}

// Ensure RS256Signer implements Signer interface.
var _ Signer[RegisteredClaims] = (*RS256Signer[RegisteredClaims])(nil)

// NewRS256Signer creates an RS256Signer from a PEM-encoded RSA private key.
//
// The key must be in PKCS#1 format (header "RSA PRIVATE KEY") or PKCS#8 format
// (header "PRIVATE KEY"). The function handles keys with escaped newlines
// (literal "\n" instead of actual newlines) and surrounding quotes, which is
// common when keys are stored in environment variables or JSON.
//
// For security, RSA keys should be at least 2048 bits. Smaller keys are accepted
// but provide inadequate security for production use.
//
// Returns an error if the PEM data is invalid, the key format is unsupported,
// or the key is not an RSA key (e.g., EC or Ed25519).
func NewRS256Signer[T Claims](privateKeyPEM string) (*RS256Signer[T], error) {
	if err := assert.NotEmpty(privateKeyPEM, "private key PEM must not be empty"); err != nil {
		return nil, err
	}

	key, err := parsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	return &RS256Signer[T]{privateKey: key}, nil
}

// Sign creates a signed JWT from the given claims.
//
// The token header is {"alg":"RS256","typ":"JWT"}. The claims are JSON-serialized
// and the signature is computed as RSA-PKCS1v15-SHA256(privateKey, SHA256(header.payload)).
//
// Returns the complete JWT in the format "header.payload.signature" where each
// segment uses base64url encoding without padding.
//
// Returns an error if JSON marshaling fails or if the RSA signing operation fails
// (which should not happen with a valid key).
func (s *RS256Signer[T]) Sign(claims T) (string, error) {
	header := map[string]string{
		"alg": "RS256",
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

	hash := sha256.Sum256([]byte(signatureInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, s.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	return signatureInput + "." + encodedSignature, nil
}

// RS256Verifier validates JSON Web Tokens signed with the RS256 algorithm
// (RSA PKCS#1 v1.5 with SHA-256).
//
// The verifier only accepts tokens with alg=RS256 in the header. Tokens with
// any other algorithm (including "none", HS256, etc.) are rejected. This
// prevents algorithm confusion attacks where an attacker might try to use
// a symmetric algorithm with the public key as the secret.
//
// The type parameter T specifies the claims type, which must implement [Claims].
// RS256Verifier is safe for concurrent use.
type RS256Verifier[T Claims] struct {
	publicKey *rsa.PublicKey
	clock     clock.Clock
}

// Ensure RS256Verifier implements Verifier interface.
var _ Verifier[RegisteredClaims] = (*RS256Verifier[RegisteredClaims])(nil)

// NewRS256Verifier creates an RS256Verifier from a PEM-encoded RSA public key.
//
// The key must be in PKIX/SPKI format (header "PUBLIC KEY"), which is the
// standard format for public keys. The function handles keys with escaped
// newlines and surrounding quotes.
//
// By default, the verifier uses the system clock for temporal claims validation.
// Pass a [clock.Clock] to override this, which is useful for testing:
//
//	clk := clock.NewTestClock(fixedTime)
//	verifier, err := NewRS256Verifier[MyClaims](publicKeyPEM, clk)
//
// Returns an error if the PEM data is invalid or the key is not an RSA public key.
func NewRS256Verifier[T Claims](publicKeyPEM string, clk ...clock.Clock) (*RS256Verifier[T], error) {
	if err := assert.NotEmpty(publicKeyPEM, "public key PEM must not be empty"); err != nil {
		return nil, err
	}

	key, err := parsePublicKey(publicKeyPEM)
	if err != nil {
		return nil, err
	}

	var c clock.Clock = clock.New()
	if len(clk) > 0 {
		c = clk[0]
	}

	return &RS256Verifier[T]{publicKey: key, clock: c}, nil
}

// Verify validates a JWT and returns the typed claims.
//
// Verification proceeds in order:
//  1. Structure: token must have exactly three dot-separated parts
//  2. Header decoding: first part must be valid base64url JSON with alg field
//  3. Algorithm check: alg must be exactly "RS256"
//  4. Signature verification: RSA-PKCS1v15-SHA256 verification with public key
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
func (v *RS256Verifier[T]) Verify(token string, at ...time.Time) (T, error) {
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

	if err := assert.Equal(header.Alg, "RS256", "unsupported algorithm"); err != nil {
		return claims, err
	}

	signatureInput := parts[0] + "." + parts[1]
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return claims, fmt.Errorf("invalid signature encoding: %w", err)
	}

	hash := sha256.Sum256([]byte(signatureInput))
	if err := rsa.VerifyPKCS1v15(v.publicKey, crypto.SHA256, hash[:], signature); err != nil {
		return claims, fmt.Errorf("invalid signature: %w", err)
	}

	payloadJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, fmt.Errorf("invalid payload encoding: %w", err)
	}

	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return claims, fmt.Errorf("invalid payload JSON: %w", err)
	}

	if err := claims.Validate(verifyAt); err != nil {
		return claims, err
	}

	return claims, nil
}

// parsePrivateKey extracts an RSA private key from PEM data.
//
// Handles both PKCS#1 ("RSA PRIVATE KEY") and PKCS#8 ("PRIVATE KEY") formats.
// Normalizes escaped newlines and strips surrounding quotes to handle keys
// from environment variables and JSON config files.
func parsePrivateKey(pemData string) (*rsa.PrivateKey, error) {
	pemData = strings.ReplaceAll(pemData, "\\n", "\n")
	pemData = strings.Trim(pemData, "\"")

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not RSA")
		}
		return rsaKey, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", block.Type)
	}
}

// parsePublicKey extracts an RSA public key from PEM data.
//
// Expects PKIX/SPKI format ("PUBLIC KEY"). Normalizes escaped newlines and
// strips surrounding quotes to handle keys from environment variables and
// JSON config files.
func parsePublicKey(pemData string) (*rsa.PublicKey, error) {
	pemData = strings.ReplaceAll(pemData, "\\n", "\n")
	pemData = strings.Trim(pemData, "\"")

	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA")
	}

	return rsaKey, nil
}
