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
// Claims validation is performed automatically by the verifier and includes:
//   - Size limits on string claims to prevent denial of service (max 255 bytes each)
//   - Temporal validation of exp (expiration) and nbf (not before) timestamps
//   - Optional issuer validation via [WithIssuer]
//   - Optional audience validation via [WithAudience]
//
// # Key Types
//
// [Signer] and [Verifier] are the main interfaces. Use [NewHS256Signer] and
// [NewHS256Verifier] for symmetric signing, or [NewRS256Signer] and [NewRS256Verifier]
// for asymmetric signing.
//
// [RegisteredClaims] contains the standard JWT claims (iss, sub, aud, exp, nbf, iat, jti).
// Embed it in your custom claims struct to automatically include these fields.
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
// Sign tokens with a signer:
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
// Verify tokens with a verifier, optionally requiring specific issuer/audience:
//
//	verifier, err := jwt.NewRS256Verifier[MyClaims](publicKeyPEM,
//	    jwt.WithIssuer("auth-service"),
//	    jwt.WithAudience("my-api"),
//	)
//	if err != nil {
//	    return err
//	}
//
//	claims, err := verifier.Verify(token)
//	if errors.Is(err, jwt.ErrTokenExpired) {
//	    // Token has expired, prompt re-authentication
//	}
//	if errors.Is(err, jwt.ErrInvalidIssuer) {
//	    // Token was issued by an untrusted issuer
//	}
//
//	// Registered claims are validated automatically.
//	// Perform any additional custom validation here:
//	if claims.TenantID == "" {
//	    return errors.New("tenant_id is required")
//	}
//
// # Error Handling
//
// [Verifier.Verify] returns specific errors for different failure conditions:
//   - [ErrTokenExpired]: token's exp claim is in the past
//   - [ErrTokenNotYetValid]: token's nbf claim is in the future
//   - [ErrInvalidIssuer]: token's iss claim doesn't match [WithIssuer] configuration
//   - [ErrInvalidAudience]: token's aud claim doesn't contain [WithAudience] value
//
// Other errors indicate malformed tokens, invalid signatures, or wrong algorithms.
// Do not distinguish between signature failures and format errors in user-facing
// messages; both should result in a generic "invalid token" response to avoid
// leaking information to attackers probing the system.
//
// # Testing
//
// Verifiers accept [WithClock] for deterministic time-based testing:
//
//	clk := clock.NewTestClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
//	verifier, _ := jwt.NewHS256Verifier[MyClaims](secret, jwt.WithClock(clk))
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
// There is no built-in clock skew tolerance. If you need leeway for clock drift
// between services, use [WithClock] with a clock that returns a time slightly in the past.
//
// Key rotation and kid (key ID) headers are not supported. For multi-key scenarios,
// maintain multiple verifiers and try each one, or use a higher-level abstraction.
//
// [github.com/golang-jwt/jwt/v5]: https://pkg.go.dev/github.com/golang-jwt/jwt/v5
package jwt
