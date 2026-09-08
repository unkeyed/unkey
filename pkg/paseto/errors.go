package paseto

import "errors"

var (
	// ErrInvalidClaims marks a payload type or value that cannot form a valid
	// PASETO JSON object.
	ErrInvalidClaims = errors.New("paseto: invalid claims")

	// ErrInvalidKey marks key material that is invalid for its PASETO purpose.
	ErrInvalidKey = errors.New("paseto: invalid key")

	// ErrInvalidToken marks a malformed token, failed authentication, or invalid
	// authenticated payload.
	ErrInvalidToken = errors.New("paseto: invalid token")
)
