package auth

import (
	"context"

	"github.com/unkeyed/unkey/pkg/zen"
)

// Resolver derives a Principal from the request for a single authentication
// scheme (root key, JWT, cookie session). Each scheme lives in its own
// file with its own tests so the Authenticator stays a dumb chain walker.
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
