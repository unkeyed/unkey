package jwt

import "time"

// Signer creates signed JSON Web Tokens from claims.
//
// The type parameter T specifies the claims type, which must implement [Claims].
// This allows compile-time type safety when working with custom claim structures.
//
// Implementations are safe for concurrent use. The signing key is captured at
// construction time and cannot be changed afterward.
//
// Define a custom claims type by embedding [RegisteredClaims]:
//
//	type MyClaims struct {
//	    jwt.RegisteredClaims
//	    TenantID string `json:"tenant_id"`
//	}
//
// The embedded [RegisteredClaims.Validate] method is promoted automatically,
// so MyClaims satisfies [Claims] without additional code. Override Validate
// only if you need custom validation beyond exp/nbf checking.
//
// Then create a signer and sign tokens:
//
//	signer, err := jwt.NewHS256Signer[MyClaims](secret)
//	if err != nil {
//	    return err
//	}
//
//	token, err := signer.Sign(MyClaims{
//	    RegisteredClaims: jwt.RegisteredClaims{
//	        Subject:   "user-123",
//	        ExpiresAt: time.Now().Add(time.Hour).Unix(),
//	        IssuedAt:  time.Now().Unix(),
//	    },
//	    TenantID: "tenant-abc",
//	})
type Signer[T Claims] interface {
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

// Verifier validates JSON Web Tokens and extracts typed claims.
//
// The type parameter T specifies the claims type, which must implement [Claims].
// During verification, the token's payload is unmarshaled into this type, and
// the [Claims.Validate] method is called to check temporal claims.
//
// Implementations are safe for concurrent use. The verification key and clock
// are captured at construction time and cannot be changed afterward.
//
// Example:
//
//	verifier, err := jwt.NewRS256Verifier[MyClaims](publicKeyPEM)
//	claims, err := verifier.Verify(token)
//	if errors.Is(err, jwt.ErrTokenExpired) {
//	    // Handle expiration
//	}
type Verifier[T Claims] interface {
	// Verify validates a JWT and returns the typed claims.
	//
	// Verification proceeds in order: structure validation (three dot-separated parts),
	// base64url decoding, JSON parsing, algorithm check, signature verification, and
	// finally claims validation via [Claims.Validate].
	//
	// The optional time parameter overrides the current time for claims validation.
	// This is primarily useful for testing; in production, omit it to use the
	// verifier's clock. If multiple times are passed, only the first is used.
	//
	// Returns [ErrTokenExpired] if the token's exp claim is in the past, and
	// [ErrTokenNotYetValid] if the nbf claim is in the future. Other errors indicate
	// structural problems (malformed token), cryptographic failures (invalid signature,
	// wrong algorithm), or JSON parsing errors.
	//
	// On error, the returned claims value is the zero value of T and should not be used.
	Verify(token string, at ...time.Time) (T, error)
}
