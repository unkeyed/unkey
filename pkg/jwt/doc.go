// Package jwt provides JSON Web Token signing and verification with algorithm-pinned
// security to prevent algorithm confusion attacks.
//
// This package implements a subset of RFC 7519 (JSON Web Token) with a security-first
// design. Rather than supporting dynamic algorithm selection from the token header,
// each verifier is constructed for a specific algorithm. This prevents the classic
// JWT vulnerability where an attacker crafts a token with alg=none or switches
// between asymmetric and symmetric algorithms to forge signatures.
//
// # Supported Algorithms
//
// RS256 (RSA PKCS#1 v1.5 with SHA-256) is recommended for service-to-service
// authentication where signing and verification happen on different systems.
// The private key signs tokens; the public key verifies them.
//
// HS256 (HMAC with SHA-256) is appropriate when both parties share a secret,
// such as a single service signing and verifying its own tokens. The same
// secret is used for both signing and verification.
//
// # Security Model
//
// The verifier checks the algorithm in the token header and rejects tokens that
// don't match the expected algorithm. An [HS256Verifier] rejects RS256 tokens and
// vice versa. Tokens with alg=none are always rejected.
//
// Signature verification uses constant-time comparison via [crypto/hmac.Equal] for
// HS256 to prevent timing attacks. RS256 uses Go's standard [crypto/rsa] verification.
//
// Claims validation checks exp (expiration) and nbf (not before) timestamps.
// The current implementation accepts tokens at exactly their exp time; stricter
// implementations may want to reject at that boundary per RFC 7519.
//
// # Key Types
//
// [Signer] and [Verifier] are the main interfaces. Use [NewHS256Signer] and
// [NewHS256Verifier] for symmetric signing, or [NewRS256Signer] and [NewRS256Verifier]
// for asymmetric signing.
//
// [RegisteredClaims] implements the standard JWT claims (iss, sub, aud, exp, nbf, iat, jti).
// Embed it in your own claims struct and implement the [Claims] interface.
//
// # Usage
//
// Define custom claims by embedding [RegisteredClaims]:
//
//	type MyClaims struct {
//	    jwt.RegisteredClaims
//	    TenantID string `json:"tenant_id"`
//	    Role     string `json:"role"`
//	}
//
// The embedded [RegisteredClaims.Validate] is promoted automatically, so MyClaims
// satisfies [Claims] without additional code. Override Validate only if you need
// custom validation beyond exp/nbf checking.
//
// Sign and verify with RS256:
//
//	signer, err := jwt.NewRS256Signer[MyClaims](privateKeyPEM)
//	if err != nil {
//	    return err
//	}
//
//	token, err := signer.Sign(MyClaims{
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        Issuer:    "auth-service",
//	        Subject:   "user-123",
//	        ExpiresAt: time.Now().Add(time.Hour).Unix(),
//	    },
//	    TenantID: "tenant-abc",
//	})
//
//	verifier, err := jwt.NewRS256Verifier[MyClaims](publicKeyPEM)
//	if err != nil {
//	    return err
//	}
//
//	claims, err := verifier.Verify(token)
//	if errors.Is(err, jwt.ErrTokenExpired) {
//	    // Token has expired, prompt re-authentication
//	}
//
// # Error Handling
//
// [Verifier.Verify] returns [ErrTokenExpired] when the token's exp claim is in the past,
// and [ErrTokenNotYetValid] when the nbf claim is in the future. Other errors indicate
// malformed tokens, invalid signatures, or wrong algorithms.
//
// Do not distinguish between signature failures and format errors in user-facing
// messages; both should result in a generic "invalid token" response to avoid
// leaking information to attackers probing the system.
//
// # Testing
//
// Verifiers accept an optional [clock.Clock] for deterministic time-based testing:
//
//	clk := clock.NewTestClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
//	verifier, _ := jwt.NewHS256Verifier[MyClaims](secret, clk)
//
//	// Token valid at this time
//	claims, err := verifier.Verify(token)
//
//	// Advance time past expiration
//	clk.Tick(2 * time.Hour)
//	_, err = verifier.Verify(token) // Returns ErrTokenExpired
//
// # Interoperability
//
// Tokens produced by this package are compatible with [github.com/golang-jwt/jwt/v5]
// and other standard JWT libraries. The Audience claim is serialized as a JSON array;
// tokens from libraries that serialize single audiences as strings will fail to parse.
//
// # Limitations
//
// This package does not validate issuer, audience, or subject claims. Applications
// must check these values after verification if they are security-relevant.
//
// There is no built-in clock skew tolerance. If you need leeway for clock drift
// between services, pass a time slightly in the past to [Verifier.Verify].
//
// Key rotation and kid (key ID) headers are not supported. For multi-key scenarios,
// maintain multiple verifiers and try each one, or use a higher-level abstraction.
//
// [github.com/golang-jwt/jwt/v5]: https://pkg.go.dev/github.com/golang-jwt/jwt/v5
package jwt
