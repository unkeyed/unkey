package jwt

import "errors"

// ErrTokenExpired indicates that the token's exp claim is in the past relative to
// the verification time. This is an expected condition for tokens that have outlived
// their validity period; callers should prompt re-authentication rather than treating
// it as a system error.
//
// The exp claim is compared using Unix seconds. A token is considered expired when
// now.Unix() > exp. Note that a token verified at exactly its exp time is currently
// accepted; this may change in future versions to align with strict RFC 7519 interpretation.
var ErrTokenExpired = errors.New("token expired")

// ErrTokenNotYetValid indicates that the token's nbf (not before) claim is in the
// future relative to the verification time. This typically means the token was
// issued for future use and cannot be accepted yet.
//
// A token is considered valid when now.Unix() >= nbf. Unlike exp, the boundary
// condition at exactly nbf is inclusive (the token is valid).
var ErrTokenNotYetValid = errors.New("token not yet valid")

// ErrInvalidIssuer indicates that the token's iss claim does not match the
// expected issuer configured via [WithIssuer].
var ErrInvalidIssuer = errors.New("invalid issuer")

// ErrInvalidAudience indicates that the token's aud claim does not contain
// the expected audience configured via [WithAudience].
var ErrInvalidAudience = errors.New("invalid audience")
