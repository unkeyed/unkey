// Package jwt scaffolds short-lived bearer JWT authentication for API handlers.
//
// The API runtime registers the resolver only when jwt_secrets is configured.
// The list is ordered for rotation: dashboards sign new proxy tokens with the
// first secret, while svc/api verifies tokens against every configured secret.
// Keep retired secrets in the list until tokens signed with them have expired.
//
// The package uses the repository's local
// [github.com/unkeyed/unkey/pkg/jwt] verifier instead of adding another JWT
// dependency. Tokens must carry standard temporal claims and the workspace,
// subject, and permission claims needed to construct a
// [github.com/unkeyed/unkey/pkg/auth/principal.Principal].
package jwt
