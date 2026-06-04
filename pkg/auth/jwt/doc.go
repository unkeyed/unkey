// Package jwt scaffolds short-lived bearer JWT authentication for API handlers.
//
// The resolver is not registered by the API runtime yet. It uses the
// repository's local [github.com/unkeyed/unkey/pkg/jwt] verifier instead of
// adding another JWT dependency. Tokens must carry standard temporal claims and
// the workspace, subject, and permission claims needed to construct an
// [github.com/unkeyed/unkey/pkg/auth.Principal].
package jwt
