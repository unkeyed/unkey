package keyauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

// Executor handles KeyAuth policy evaluation by wrapping the existing KeyService.
type Executor struct {
	keyService keys.KeyService
	clock      clock.Clock
}

// New creates a new KeyAuth policy executor.
func New(keyService keys.KeyService, clk clock.Clock) *Executor {
	return &Executor{
		keyService: keyService,
		clock:      clk,
	}
}

// Execute evaluates a KeyAuth policy against the incoming request.
// It extracts the API key, verifies it using KeyService, and returns a Principal on success.
func (e *Executor) Execute(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	cfg *frontlinev1.KeyAuth,
) (*principal.Principal, error) {
	rawKey := extractKey(req, cfg.GetLocations())
	if rawKey == "" {
		return nil, fault.New("missing API key",
			fault.Code(codes.Frontline.Auth.MissingCredentials.URN()),
			fault.Internal("no API key found in request"),
			fault.Public("Authentication required. Please provide a valid API key."),
		)
	}

	keyHash := hash.Sha256(rawKey)
	verifier, logFn, err := e.keyService.Get(ctx, sess, keyHash)
	defer logFn()
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal("key lookup failed"),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	// Fail fast on states that verification cannot recover from (not found,
	// disabled, expired, workspace disabled, etc.) before spending a credit.
	if verifier.Status != keys.StatusValid {
		return nil, fault.New("invalid API key",
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal("key status: "+string(verifier.Status)),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	if !keyspaceAllowed(verifier.Key.KeyAuthID, cfg.GetKeySpaceIds()) {
		return nil, fault.New("key does not belong to expected key space",
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal(fmt.Sprintf("key belongs to key space %s, expected one of %s", verifier.Key.KeyAuthID, strings.Join(cfg.GetKeySpaceIds(), ","))),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	verifyOpts := []keys.VerifyOption{keys.WithCredits(1)}
	if pq := cfg.GetPermissionQuery(); pq != "" {
		query, err := rbac.ParseQuery(pq)
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.Frontline.Internal.InvalidConfiguration.URN()),
				fault.Internal("invalid permission query: "+pq),
				fault.Public("Service configuration error."),
			)
		}
		verifyOpts = append(verifyOpts, keys.WithPermissions(query))
	}

	if err := verifier.Verify(ctx, verifyOpts...); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("verification error"),
			fault.Public("An internal error occurred during authentication."),
		)
	}

	// Write rate limit headers before checking status so they're present
	// on both success (2xx) and rate-limited (429) responses.
	writeRateLimitHeaders(sess.ResponseWriter(), verifier.RatelimitResults, e.clock)

	switch verifier.Status {
	case keys.StatusValid:
		// OK
	case keys.StatusInsufficientPermissions:
		return nil, fault.New("insufficient permissions",
			fault.Code(codes.Frontline.Auth.InsufficientPermissions.URN()),
			fault.Internal("key lacks required permissions"),
			fault.Public("Access denied. The API key does not have the required permissions."),
		)
	case keys.StatusRateLimited:
		return nil, fault.New("rate limited",
			fault.Code(codes.Frontline.Auth.RateLimited.URN()),
			fault.Internal("auto-applied rate limit exceeded"),
			fault.Public("Rate limit exceeded. Please try again later."),
		)
	case keys.StatusUsageExceeded:
		return nil, fault.New("usage exceeded",
			fault.Code(codes.Frontline.Auth.RateLimited.URN()),
			fault.Internal("usage limit exceeded"),
			fault.Public("Usage limit exceeded. Please try again later."),
		)
	case keys.StatusNotFound, keys.StatusDisabled, keys.StatusExpired,
		keys.StatusForbidden, keys.StatusWorkspaceDisabled, keys.StatusWorkspaceNotFound:
		return nil, fault.New("key verification failed",
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal("post-verification status: "+string(verifier.Status)),
			fault.Public("Authentication failed."),
		)
	}

	p, err := principal.KeyPrincipalFromVerifier(verifier)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Internal("failed to build principal"),
			fault.Public("An internal error occurred during authentication."),
		)
	}
	return p, nil
}

// keyspaceAllowed reports whether the key's keyspace is in the policy's
// allowlist.
func keyspaceAllowed(keyspaceID string, allowed []string) bool {
	for _, id := range allowed {
		if keyspaceID == id {
			return true
		}
	}
	return false
}
