package engine

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

// KeyAuthExecutor handles KeyAuth policy evaluation by wrapping the existing KeyService.
type KeyAuthExecutor struct {
	keyService keys.KeyService
	clock      clock.Clock
}

// Execute evaluates a KeyAuth policy against the incoming request. It
// extracts the API key, verifies it through KeyService, writes rate limit
// headers, and returns a Principal on success.
func (e *KeyAuthExecutor) Execute(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	cfg *sentinelv1.KeyAuth,
) (*Principal, error) {
	rawKey := extractKey(req, cfg.GetLocations())
	if rawKey == "" {
		return nil, fault.New("missing API key",
			fault.Code(codes.Sentinel.Auth.MissingCredentials.URN()),
			fault.Internal("no API key found in request"),
			fault.Public("Authentication required. Please provide a valid API key."),
		)
	}

	keyHash := hash.Sha256(rawKey)
	verifier, logFn, err := e.keyService.Get(ctx, sess, keyHash)
	defer logFn()
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal("key lookup failed"),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	// Fail fast on states that verification cannot recover from (not found,
	// disabled, expired, workspace disabled, etc.) before spending a credit.
	if verifier.Status != keys.StatusValid {
		return nil, fault.New("invalid API key",
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal("key status: "+string(verifier.Status)),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	if !keyspaceAllowed(verifier.Key.KeyAuthID, cfg.GetKeySpaceIds()) {
		return nil, fault.New("key does not belong to expected key space",
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal(fmt.Sprintf("key belongs to key space %s, expected one of %s", verifier.Key.KeyAuthID, strings.Join(cfg.GetKeySpaceIds(), ","))),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	verifyOpts := []keys.VerifyOption{keys.WithCredits(1)}
	if pq := cfg.GetPermissionQuery(); pq != "" {
		query, err := rbac.ParseQuery(pq)
		if err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
				fault.Internal("invalid permission query: "+pq),
				fault.Public("Service configuration error."),
			)
		}
		verifyOpts = append(verifyOpts, keys.WithPermissions(query))
	}

	if err := verifier.Verify(ctx, verifyOpts...); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
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
			fault.Code(codes.Sentinel.Auth.InsufficientPermissions.URN()),
			fault.Internal("key lacks required permissions"),
			fault.Public("Access denied. The API key does not have the required permissions."),
		)
	case keys.StatusRateLimited:
		return nil, fault.New("rate limited",
			fault.Code(codes.Sentinel.Auth.RateLimited.URN()),
			fault.Internal("auto-applied rate limit exceeded"),
			fault.Public("Rate limit exceeded. Please try again later."),
		)
	case keys.StatusUsageExceeded:
		return nil, fault.New("usage exceeded",
			fault.Code(codes.Sentinel.Auth.RateLimited.URN()),
			fault.Internal("usage limit exceeded"),
			fault.Public("Usage limit exceeded. Please try again later."),
		)
	case keys.StatusNotFound, keys.StatusDisabled, keys.StatusExpired,
		keys.StatusForbidden, keys.StatusWorkspaceDisabled, keys.StatusWorkspaceNotFound:
		// These should have been caught by the pre-verify status check, but
		// handle them here for exhaustiveness.
		return nil, fault.New("key verification failed",
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal("post-verification status: "+string(verifier.Status)),
			fault.Public("Authentication failed."),
		)
	}

	principal, err := keyPrincipalFromVerifier(verifier)
	if err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("failed to build principal"),
			fault.Public("An internal error occurred during authentication."),
		)
	}
	return principal, nil
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

// writeRateLimitHeaders sets standard rate limit headers on the response.
// When multiple rate limits exist, it uses the most restrictive one (lowest remaining).
func writeRateLimitHeaders(w http.ResponseWriter, results map[string]keys.RatelimitConfigAndResult, clk clock.Clock) {
	if len(results) == 0 {
		return
	}

	// Find the most restrictive rate limit (lowest remaining).
	var mostRestrictive *keys.RatelimitConfigAndResult
	for _, r := range results {
		if r.Response == nil {
			continue
		}

		if mostRestrictive == nil || r.Response.Remaining < mostRestrictive.Response.Remaining {
			rCopy := r
			mostRestrictive = &rCopy
		}
	}

	if mostRestrictive == nil {
		return
	}

	resp := mostRestrictive.Response
	h := w.Header()
	h.Set("X-RateLimit-Limit", strconv.FormatInt(resp.Limit, 10))
	h.Set("X-RateLimit-Remaining", strconv.FormatInt(resp.Remaining, 10))
	h.Set("X-RateLimit-Reset", strconv.FormatInt(resp.Reset.Unix(), 10))

	if !resp.Success {
		retryAfter := math.Ceil(resp.Reset.Sub(clk.Now()).Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}

		h.Set("Retry-After", strconv.FormatInt(int64(retryAfter), 10))
	}
}
