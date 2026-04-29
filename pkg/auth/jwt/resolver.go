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

// Resolver matches any bearer shaped like a JWT (three dot-separated
// segments) and verifies it as an HS256 token against secret. Construct
// it only when a secret is configured; without one, JWTs must not be
// accepted.
type Resolver struct {
	secret []byte
}

// NewResolver builds a JWT auth.Resolver bound to the given HMAC secret.
func NewResolver(secret []byte) *Resolver {
	return &Resolver{secret: secret}
}

// Resolve returns (nil, _, nil) when the bearer is missing or doesn't
// look like a JWT, leaving it for other resolvers. Anything else is
// claimed and verified; verification failure terminates the chain with
// a Malformed-credential fault.
func (r *Resolver) Resolve(ctx context.Context, sess *zen.Session) (*auth.Principal, auth.Emit, error) {
	bearer, err := zen.Bearer(sess)
	if err != nil || len(strings.Split(bearer, ".")) != 3 {
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
