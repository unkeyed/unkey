package auth

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
)

// Resolver tries to derive a Principal from the request. Each authentication
// scheme (root key, JWT, cookie session) is one Resolver, lives in its own
// file with its own tests, and the dispatcher stays a dumb chain walker
// that runs cross-cutting checks (rate limiting, observability) in exactly
// one place.
//
// The three return states:
//
//   - (nil, _, nil): not my scheme; try the next resolver.
//   - (p, emit, nil): success; principal is fully resolved.
//   - (nil, _, err): my scheme matched but the credential is invalid;
//     stop the chain and surface the error.
//
// Returning (nil, _, nil) on a malformed credential of your scheme is a
// bug: it lets the next resolver run and produce a misleading error.
type Resolver interface {
	Resolve(ctx context.Context, sess *zen.Session) (*Principal, Emit, error)
}
