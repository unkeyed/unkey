package jwt

import (
	"context"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

// Resolver matches any bearer that does NOT carry the "unkey_" root-key
// prefix and verifies it as an HS256 JWT against secret. Construct it only
// when a secret is configured; without one, JWTs must not be accepted.
type Resolver struct {
	secret []byte
}

// NewResolver builds a JWT auth.Resolver bound to the given HMAC secret.
func NewResolver(secret []byte) *Resolver {
	return &Resolver{secret: secret}
}

// Try returns (nil, _, nil) when the bearer is missing or carries the
// "unkey_" root-key prefix, leaving those for the root-key resolver.
// Anything else is claimed and verified; verification failure terminates
// the chain with a Malformed-credential fault.
func (r *Resolver) Try(ctx context.Context, sess *zen.Session) (*auth.Principal, auth.Emit, error) {
	bearer, err := zen.Bearer(sess)
	if err != nil || strings.HasPrefix(bearer, "unkey_") {
		return nil, nil, nil
	}
	p, verr := Verify(bearer, r.secret)
	if verr != nil {
		logger.Warn("jwt verification failed",
			"error", verr.Error(),
			"request_id", sess.RequestID(),
		)
		return nil, auth.EmptyEmit, fault.Wrap(verr,
			fault.Code(codes.Auth.Authentication.Malformed.URN()),
			fault.Internal(verr.Error()),
			fault.Public("The provided token is invalid."),
		)
	}
	return p, auth.EmptyEmit, nil
}
