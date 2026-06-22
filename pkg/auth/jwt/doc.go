// Package jwt scaffolds short-lived bearer JWT authentication for API handlers.
//
// The API runtime registers one resolver for each ordered auth entry. Entries
// with type="jwt" verify tokens from one configured issuer using exactly one key
// source. The secrets source verifies HS256 tokens with an ordered secret list
// for rotation: dashboards sign new proxy tokens with the first secret, while
// svc/api verifies tokens against every configured secret. The JWKS source
// fetches RS256 RSA signing keys from a JWKS endpoint lazily on first use,
// selects the key named by the token's kid header, and refetches the set when
// a token's signing key may be missing from it, so signing-key rotations are
// picked up without a restart. Malformed JWKS entries are skipped rather than
// failing the fetch.
//
// The package uses the repository's local
// [github.com/unkeyed/unkey/pkg/jwt] verifier instead of adding another JWT
// dependency. Tokens must carry standard temporal claims and enough identity,
// organization, and permission data to construct a
// [github.com/unkeyed/unkey/pkg/auth/principal.Principal].
package jwt
