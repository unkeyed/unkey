---
title: jwt
description: "provides JSON Web Token signing and verification with algorithm-pinned"
---

Package jwt provides JSON Web Token signing and verification with algorithm-pinned security to prevent algorithm confusion attacks.

This package implements a subset of RFC 7519 (JSON Web Token) with a security-first design. Rather than supporting dynamic algorithm selection from the token header, each verifier is constructed for a specific algorithm. This prevents the classic JWT vulnerability where an attacker crafts a token with alg=none or switches between asymmetric and symmetric algorithms to forge signatures.

### Supported Algorithms

RS256 (RSA PKCS#1 v1.5 with SHA-256) is recommended for service-to-service authentication where signing and verification happen on different systems. The private key signs tokens; the public key verifies them.

HS256 (HMAC with SHA-256) is appropriate when both parties share a secret, such as a single service signing and verifying its own tokens. The same secret is used for both signing and verification.

### Security Model

The verifier checks the algorithm in the token header and rejects tokens that don't match the expected algorithm. An \[HS256Verifier] rejects RS256 tokens and vice versa. Tokens with alg=none are always rejected.

Signature verification uses constant-time comparison via [crypto/hmac.Equal](/crypto/hmac#Equal) for HS256 to prevent timing attacks. RS256 uses Go's standard [crypto/rsa](/crypto/rsa) verification.

Claims validation is performed automatically by the verifier and includes:

  - Size limits on string claims to prevent denial of service (max 255 bytes each)
  - Temporal validation of exp (expiration) and nbf (not before) timestamps
  - Optional issuer validation via \[WithIssuer]
  - Optional audience validation via \[WithAudience]

### Key Types

\[Signer] and \[Verifier] are the main interfaces. Use \[NewHS256Signer] and \[NewHS256Verifier] for symmetric signing, or \[NewRS256Signer] and \[NewRS256Verifier] for asymmetric signing.

\[RegisteredClaims] contains the standard JWT claims (iss, sub, aud, exp, nbf, iat, jti). Embed it in your custom claims struct to automatically include these fields.

### Usage

Define custom claims by embedding \[RegisteredClaims]:

	type MyClaims struct {
	    jwt.RegisteredClaims
	    TenantID string `json:"tenant_id"`
	    Role     string `json:"role"`
	}

Sign tokens with a signer:

	signer, err := jwt.NewRS256Signer[MyClaims](privateKeyPEM)
	if err != nil {
	    return err
	}

	token, err := signer.Sign(MyClaims{
	    RegisteredClaims: jwt.RegisteredClaims{
	        Issuer:    "auth-service",
	        Subject:   "user-123",
	        ExpiresAt: time.Now().Add(time.Hour).Unix(),
	    },
	    TenantID: "tenant-abc",
	})

Verify tokens with a verifier, optionally requiring specific issuer/audience:

	verifier, err := jwt.NewRS256Verifier[MyClaims](publicKeyPEM,
	    jwt.WithIssuer("auth-service"),
	    jwt.WithAudience("my-api"),
	)
	if err != nil {
	    return err
	}

	claims, err := verifier.Verify(token)
	if errors.Is(err, jwt.ErrTokenExpired) {
	    // Token has expired, prompt re-authentication
	}
	if errors.Is(err, jwt.ErrInvalidIssuer) {
	    // Token was issued by an untrusted issuer
	}

	// Registered claims are validated automatically.
	// Perform any additional custom validation here:
	if claims.TenantID == "" {
	    return errors.New("tenant_id is required")
	}

### Error Handling

\[Verifier.Verify] returns specific errors for different failure conditions:

  - \[ErrTokenExpired]: token's exp claim is in the past
  - \[ErrTokenNotYetValid]: token's nbf claim is in the future
  - \[ErrInvalidIssuer]: token's iss claim doesn't match \[WithIssuer] configuration
  - \[ErrInvalidAudience]: token's aud claim doesn't contain \[WithAudience] value

Other errors indicate malformed tokens, invalid signatures, or wrong algorithms. Do not distinguish between signature failures and format errors in user-facing messages; both should result in a generic "invalid token" response to avoid leaking information to attackers probing the system.

### Testing

Verifiers accept \[WithClock] for deterministic time-based testing:

	clk := clock.NewTestClock(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))
	verifier, _ := jwt.NewHS256Verifier[MyClaims](secret, jwt.WithClock(clk))

	// Token valid at this time
	claims, err := verifier.Verify(token)

	// Advance time past expiration
	clk.Tick(2 * time.Hour)
	_, err = verifier.Verify(token) // Returns ErrTokenExpired

### Interoperability

Tokens produced by this package are compatible with [github.com/golang-jwt/jwt/v5](https://pkg.go.dev/github.com/golang-jwt/jwt/v5) and other standard JWT libraries. The Audience claim is serialized as a JSON array; tokens from libraries that serialize single audiences as strings will fail to parse.

### Limitations

There is no built-in clock skew tolerance. If you need leeway for clock drift between services, use \[WithClock] with a clock that returns a time slightly in the past.

Key rotation and kid (key ID) headers are not supported. For multi-key scenarios, maintain multiple verifiers and try each one, or use a higher-level abstraction.

## Constants

Maximum sizes for registered claim values in bytes. These limits protect against denial of service attacks using oversized tokens.
```go
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
```


## Variables

ErrInvalidAudience indicates that the token's aud claim does not contain the expected audience configured via \[WithAudience].
```go
var ErrInvalidAudience = errors.New("invalid audience")
```

ErrInvalidIssuer indicates that the token's iss claim does not match the expected issuer configured via \[WithIssuer].
```go
var ErrInvalidIssuer = errors.New("invalid issuer")
```

ErrTokenExpired indicates that the token's exp claim is in the past relative to the verification time. This is an expected condition for tokens that have outlived their validity period; callers should prompt re-authentication rather than treating it as a system error.

The exp claim is compared using Unix seconds. A token is considered expired when now.Unix() > exp. Note that a token verified at exactly its exp time is currently accepted; this may change in future versions to align with strict RFC 7519 interpretation.
```go
var ErrTokenExpired = errors.New("token expired")
```

ErrTokenNotYetValid indicates that the token's nbf (not before) claim is in the future relative to the verification time. This typically means the token was issued for future use and cannot be accepted yet.

A token is considered valid when now.Unix() >= nbf. Unlike exp, the boundary condition at exactly nbf is inclusive (the token is valid).
```go
var ErrTokenNotYetValid = errors.New("token not yet valid")
```


## Functions


## Types

### type HS256Signer

```go
type HS256Signer[T any] struct {
	secret []byte
}
```

HS256Signer creates signed JSON Web Tokens using the HS256 algorithm (HMAC with SHA-256).

HS256 is a symmetric algorithm: the same secret is used for both signing and verification. Use this when the signing and verifying parties can securely share a secret, such as a single service signing tokens for its own use.

For scenarios where the signing party should not be able to verify (or vice versa), use \[RS256Signer] with asymmetric keys instead.

The type parameter T must embed \[RegisteredClaims] to include standard JWT fields. HS256Signer is safe for concurrent use; the secret is captured at construction.

#### func NewHS256Signer

```go
func NewHS256Signer[T any](secret []byte) (*HS256Signer[T], error)
```

NewHS256Signer creates an HS256Signer with the given secret key.

The secret should be a cryptographically random value. While any non-empty secret is accepted, secrets shorter than 32 bytes (256 bits) provide less security than the underlying SHA-256 hash and are not recommended for production use.

The secret slice is stored directly; the caller should not modify it after passing it to this function.

Returns an error if the secret is nil or empty.

#### func (HS256Signer) Sign

```go
func (s *HS256Signer[T]) Sign(claims T) (string, error)
```

Sign creates a signed JWT from the given claims.

The token header is {"alg":"HS256","typ":"JWT"}. The claims are JSON-serialized and the signature is computed as HMAC-SHA256(secret, base64url(header) + "." + base64url(claims)).

Returns the complete JWT in the format "header.payload.signature" where each segment uses base64url encoding without padding.

Returns an error only if JSON marshaling fails, which indicates the claims contain types that cannot be serialized (channels, functions, etc.).

### type HS256Verifier

```go
type HS256Verifier[T any] struct {
	secret []byte
	clock  clock.Clock
	config verifyConfig
}
```

HS256Verifier validates JSON Web Tokens signed with the HS256 algorithm (HMAC with SHA-256).

The verifier only accepts tokens with alg=HS256 in the header. Tokens with any other algorithm (including "none", RS256, etc.) are rejected. This prevents algorithm confusion attacks where an attacker might try to use a different algorithm to forge a valid signature.

Signature comparison uses [crypto/hmac.Equal](/crypto/hmac#Equal) which is constant-time to prevent timing attacks.

The type parameter T must embed \[RegisteredClaims] to include standard JWT fields. HS256Verifier is safe for concurrent use.

#### func NewHS256Verifier

```go
func NewHS256Verifier[T any](secret []byte, opts ...VerifyOption) (*HS256Verifier[T], error)
```

NewHS256Verifier creates an HS256Verifier with the given secret key.

The secret must match the secret used by the corresponding \[HS256Signer]. Tokens signed with a different secret will fail verification.

Options can be used to configure verification behavior:

	verifier, err := NewHS256Verifier[MyClaims](secret,
	    jwt.WithIssuer("https://auth.example.com"),
	    jwt.WithAudience("my-service"),
	)

Returns an error if the secret is nil or empty.

#### func (HS256Verifier) Verify

```go
func (v *HS256Verifier[T]) Verify(token string, at ...time.Time) (T, error)
```

Verify validates a JWT and returns the typed claims.

Verification proceeds in order:

 1. Structure: token must have exactly three dot-separated parts
 2. Header decoding: first part must be valid base64url JSON with alg field
 3. Algorithm check: alg must be exactly "HS256"
 4. Signature verification: HMAC-SHA256 of header.payload must match
 5. Payload decoding: second part must be valid base64url JSON
 6. Claims validation: \[Claims.Validate] is called with the verification time

The optional time parameter overrides the clock for claims validation. This is useful for testing tokens at specific times. In production, omit this parameter. If multiple times are passed, only the first is used.

Returns \[ErrTokenExpired] if the token's exp claim is in the past, and \[ErrTokenNotYetValid] if the nbf claim is in the future. Other errors indicate malformed tokens, wrong algorithms, or invalid signatures.

On any error, the returned claims value is the zero value of T. Do not use partial claims from failed verification.

### type RS256Signer

```go
type RS256Signer[T any] struct {
	privateKey *rsa.PrivateKey
}
```

RS256Signer creates signed JSON Web Tokens using the RS256 algorithm (RSA PKCS#1 v1.5 with SHA-256).

RS256 is an asymmetric algorithm: tokens are signed with a private key and verified with the corresponding public key. Use this for service-to-service authentication where the signing service should be the only entity capable of creating valid tokens, while multiple services can verify them.

For simpler scenarios where the same service signs and verifies, consider \[HS256Signer] which uses symmetric keys.

The type parameter T must embed \[RegisteredClaims] to include standard JWT fields. RS256Signer is safe for concurrent use; the private key is captured at construction.

#### func NewRS256Signer

```go
func NewRS256Signer[T any](privateKeyPEM string) (*RS256Signer[T], error)
```

NewRS256Signer creates an RS256Signer from a PEM-encoded RSA private key.

The key must be in PKCS#1 format (header "RSA PRIVATE KEY") or PKCS#8 format (header "PRIVATE KEY"). The function handles keys with escaped newlines (literal "\\n" instead of actual newlines) and surrounding quotes, which is common when keys are stored in environment variables or JSON.

For security, RSA keys should be at least 2048 bits. Smaller keys are accepted but provide inadequate security for production use.

Returns an error if the PEM data is invalid, the key format is unsupported, or the key is not an RSA key (e.g., EC or Ed25519).

#### func (RS256Signer) Sign

```go
func (s *RS256Signer[T]) Sign(claims T) (string, error)
```

Sign creates a signed JWT from the given claims.

The token header is {"alg":"RS256","typ":"JWT"}. The claims are JSON-serialized and the signature is computed as RSA-PKCS1v15-SHA256(privateKey, SHA256(header.payload)).

Returns the complete JWT in the format "header.payload.signature" where each segment uses base64url encoding without padding.

Returns an error if JSON marshaling fails or if the RSA signing operation fails (which should not happen with a valid key).

### type RS256Verifier

```go
type RS256Verifier[T any] struct {
	publicKey *rsa.PublicKey
	clock     clock.Clock
	config    verifyConfig
}
```

RS256Verifier validates JSON Web Tokens signed with the RS256 algorithm (RSA PKCS#1 v1.5 with SHA-256).

The verifier only accepts tokens with alg=RS256 in the header. Tokens with any other algorithm (including "none", HS256, etc.) are rejected. This prevents algorithm confusion attacks where an attacker might try to use a symmetric algorithm with the public key as the secret.

The type parameter T must embed \[RegisteredClaims] to include standard JWT fields. RS256Verifier is safe for concurrent use.

#### func NewRS256Verifier

```go
func NewRS256Verifier[T any](publicKeyPEM string, opts ...VerifyOption) (*RS256Verifier[T], error)
```

NewRS256Verifier creates an RS256Verifier from a PEM-encoded RSA public key.

The key must be in PKIX/SPKI format (header "PUBLIC KEY"), which is the standard format for public keys. The function handles keys with escaped newlines and surrounding quotes.

Options can be used to configure verification behavior:

	verifier, err := NewRS256Verifier[MyClaims](publicKeyPEM,
	    jwt.WithIssuer("https://auth.example.com"),
	    jwt.WithAudience("my-service"),
	)

Returns an error if the PEM data is invalid or the key is not an RSA public key.

#### func (RS256Verifier) Verify

```go
func (v *RS256Verifier[T]) Verify(token string, at ...time.Time) (T, error)
```

Verify validates a JWT and returns the typed claims.

Verification proceeds in order:

 1. Structure: token must have exactly three dot-separated parts
 2. Header decoding: first part must be valid base64url JSON with alg field
 3. Algorithm check: alg must be exactly "RS256"
 4. Signature verification: RSA-PKCS1v15-SHA256 verification with public key
 5. Payload decoding: second part must be valid base64url JSON
 6. Claims validation: \[Claims.Validate] is called with the verification time

The optional time parameter overrides the clock for claims validation. This is useful for testing tokens at specific times. In production, omit this parameter. If multiple times are passed, only the first is used.

Returns \[ErrTokenExpired] if the token's exp claim is in the past, and \[ErrTokenNotYetValid] if the nbf claim is in the future. Other errors indicate malformed tokens, wrong algorithms, or invalid signatures.

On any error, the returned claims value is the zero value of T. Do not use partial claims from failed verification.

### type RegisteredClaims

```go
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
```

RegisteredClaims contains the standard JWT registered claim names as defined in RFC 7519 Section 4.1. Embed this struct in your custom claims type to automatically include the standard fields with correct JSON serialization.

All fields are optional per the JWT specification. Zero values are treated as "not set" and are omitted from JSON output (via omitempty) and skipped during validation.

Example:

	type SessionClaims struct {
	    jwt.RegisteredClaims
	    SessionID string `json:"sid"`
	}

	claims := SessionClaims{
	    RegisteredClaims: jwt.RegisteredClaims{
	        Subject:   "user-123",
	        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	        IssuedAt:  time.Now().Unix(),
	    },
	    SessionID: "sess-abc",
	}

### type Signer

```go
type Signer[T any] interface {
	// Sign creates a signed JWT from the given claims.
	//
	// The claims parameter should be a struct embedding [RegisteredClaims] with any
	// additional application-specific fields. Set [RegisteredClaims.ExpiresAt] to a
	// Unix timestamp to control token expiration:
	//
	//	claims := MyClaims{
	//	    RegisteredClaims: jwt.RegisteredClaims{
	//	        Issuer:    "auth-service",
	//	        Subject:   userID,
	//	        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
	//	        IssuedAt:  time.Now().Unix(),
	//	    },
	//	    Role: "admin",
	//	}
	//	token, err := signer.Sign(claims)
	//
	// Returns the complete JWT as a string in the format "header.payload.signature"
	// where each segment is base64url-encoded without padding. The header contains
	// the algorithm (alg) and type (typ) fields; the payload contains the JSON-serialized
	// claims; the signature is algorithm-specific.
	//
	// Sign does not validate the claims before signing. It is the caller's responsibility
	// to ensure exp, nbf, and other temporal claims are set correctly. Signing a token
	// with an already-expired exp will succeed, but the token will fail verification.
	//
	// Returns an error if JSON marshaling fails for the claims. This typically indicates
	// a claims struct with fields that cannot be serialized (channels, functions, etc.).
	Sign(claims T) (string, error)
}
```

Signer creates signed JSON Web Tokens from claims.

The type parameter T should embed \[RegisteredClaims] for standard JWT fields. This allows compile-time type safety when working with custom claim structures.

Implementations are safe for concurrent use. The signing key is captured at construction time and cannot be changed afterward.

Define a custom claims type by embedding \[RegisteredClaims]:

	type MyClaims struct {
	    jwt.RegisteredClaims
	    TenantID string `json:"tenant_id"`
	}

Then create a signer and sign tokens:

	signer, err := jwt.NewHS256Signer[MyClaims](secret)
	if err != nil {
	    return err
	}

	token, err := signer.Sign(MyClaims{
	    RegisteredClaims: jwt.RegisteredClaims{
	        Subject:   "user-123",
	        ExpiresAt: time.Now().Add(time.Hour).Unix(),
	        IssuedAt:  time.Now().Unix(),
	    },
	    TenantID: "tenant-abc",
	})

### type Verifier

```go
type Verifier[T any] interface {
	// Verify validates a JWT and returns the typed claims.
	//
	// Verification proceeds in order:
	//  1. Structure validation (three dot-separated parts)
	//  2. Base64url decoding of header and payload
	//  3. Algorithm check (must match verifier's algorithm)
	//  4. Signature verification
	//  5. JSON parsing into claims type T
	//  6. Registered claims validation (exp, nbf, size limits, iss, aud)
	//
	// The optional time parameter overrides the clock for claims validation. This is
	// useful for testing; in production, configure the clock via [WithClock] at
	// construction time. If multiple times are passed, only the first is used.
	//
	// Returns specific errors for validation failures:
	//   - [ErrTokenExpired]: exp claim is in the past
	//   - [ErrTokenNotYetValid]: nbf claim is in the future
	//   - [ErrInvalidIssuer]: iss claim doesn't match [WithIssuer] configuration
	//   - [ErrInvalidAudience]: aud claim doesn't include [WithAudience] value
	//
	// Other errors indicate structural problems (malformed token), cryptographic
	// failures (invalid signature, wrong algorithm), or JSON parsing errors.
	//
	// On error, the returned claims value is the zero value of T and should not be used.
	Verify(token string, at ...time.Time) (T, error)
}
```

Verifier validates JSON Web Tokens and extracts typed claims.

The type parameter T should embed \[RegisteredClaims] for standard JWT fields. During verification, the token's payload is unmarshaled into this type, and the registered claims are validated automatically (exp, nbf, size limits, and optionally iss/aud via \[WithIssuer] and \[WithAudience]).

Implementations are safe for concurrent use. The verification key and options are captured at construction time and cannot be changed afterward.

Example:

	verifier, err := jwt.NewRS256Verifier[MyClaims](publicKeyPEM,
	    jwt.WithIssuer("https://auth.example.com"),
	    jwt.WithAudience("my-service"),
	)
	claims, err := verifier.Verify(token)
	if errors.Is(err, jwt.ErrTokenExpired) {
	    // Handle expiration
	}
	// Perform any additional custom validation on claims

### type VerifyOption

```go
type VerifyOption func(*verifyConfig)
```

VerifyOption configures token verification behavior.

#### func WithAudience

```go
func WithAudience(aud string) VerifyOption
```

WithAudience requires tokens to include the specified audience in their aud claim. If the token's audience does not contain this value, verification fails with \[ErrInvalidAudience].

#### func WithClock

```go
func WithClock(c clock.Clock) VerifyOption
```

WithClock sets a custom clock for temporal claims validation. This is primarily useful for testing with fixed or controlled time.

#### func WithIssuer

```go
func WithIssuer(iss string) VerifyOption
```

WithIssuer requires tokens to have the specified issuer (iss claim). If the token's issuer does not match, verification fails with \[ErrInvalidIssuer].

