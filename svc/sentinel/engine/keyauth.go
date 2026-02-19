package engine

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

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

// Execute evaluates a KeyAuth policy against the incoming request.
// It extracts the API key, verifies it using KeyService, and returns a Principal on success.
func (e *KeyAuthExecutor) Execute(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	cfg *sentinelv1.KeyAuth,
) (*sentinelv1.Principal, error) {
	rawKey := extractKey(req, cfg.GetLocations())
	if rawKey == "" {
		if cfg.GetAllowAnonymous() {
			return nil, nil
		}

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

	// Check basic validation (not found, disabled, expired, workspace disabled, etc.)
	if verifier.Status != keys.StatusValid {
		return nil, fault.New("invalid API key",
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal("key status: "+string(verifier.Status)),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	// Verify the key belongs to the expected key space
	if cfg.GetKeySpaceId() != "" && verifier.Key.KeyAuthID != cfg.GetKeySpaceId() {
		return nil, fault.New("key does not belong to expected key space",
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal("key belongs to key space "+verifier.Key.KeyAuthID+", expected "+cfg.GetKeySpaceId()),
			fault.Public("Authentication failed. The provided API key is invalid."),
		)
	}

	// Build verify options
	var verifyOpts []keys.VerifyOption

	if pq := cfg.GetPermissionQuery(); pq != "" {
		query, parseErr := rbac.ParseQuery(pq)
		if parseErr != nil {
			return nil, fault.Wrap(parseErr,
				fault.Code(codes.Sentinel.Internal.InvalidConfiguration.URN()),
				fault.Internal("invalid permission query: "+pq),
				fault.Public("Service configuration error."),
			)
		}

		verifyOpts = append(verifyOpts, keys.WithPermissions(query))
	}

	// Deduct 1 credit per request by default
	verifyOpts = append(verifyOpts, keys.WithCredits(1))

	verifyErr := verifier.Verify(ctx, verifyOpts...)
	if verifyErr != nil {
		return nil, fault.Wrap(verifyErr,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("verification error"),
			fault.Public("An internal error occurred during authentication."),
		)
	}

	// Write rate limit headers before checking status so they're present
	// on both success (2xx) and rate-limited (429) responses.
	writeRateLimitHeaders(sess.ResponseWriter(), verifier.RatelimitResults, e.clock)

	// Check post-verification status
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
		// These should have been caught by the pre-verify status check above,
		// but handle them here for exhaustiveness.
		return nil, fault.New("key verification failed",
			fault.Code(codes.Sentinel.Auth.InvalidKey.URN()),
			fault.Internal("post-verification status: "+string(verifier.Status)),
			fault.Public("Authentication failed."),
		)
	}

	// Build the principal
	subject := verifier.Key.ID
	if verifier.Key.ExternalID.Valid && verifier.Key.ExternalID.String != "" {
		subject = verifier.Key.ExternalID.String
	}

	claims := map[string]string{
		"key_id":       verifier.Key.ID,
		"key_space_id": verifier.Key.KeyAuthID,
		"api_id":       verifier.Key.ApiID,
		"workspace_id": verifier.Key.WorkspaceID,
	}
	if verifier.Key.Name.Valid && verifier.Key.Name.String != "" {
		claims["name"] = verifier.Key.Name.String
	}
	if verifier.Key.IdentityID.Valid && verifier.Key.IdentityID.String != "" {
		claims["identity_id"] = verifier.Key.IdentityID.String
	}
	if verifier.Key.ExternalID.Valid && verifier.Key.ExternalID.String != "" {
		claims["external_id"] = verifier.Key.ExternalID.String
	}
	if verifier.Key.Meta.Valid && verifier.Key.Meta.String != "" {
		claims["meta"] = verifier.Key.Meta.String
	}
	if verifier.Key.Expires.Valid {
		claims["expires"] = verifier.Key.Expires.Time.Format(time.RFC3339)
	}

	//nolint:exhaustruct
	return &sentinelv1.Principal{
		Subject: subject,
		Type:    sentinelv1.PrincipalType_PRINCIPAL_TYPE_API_KEY,
		Claims:  claims,
	}, nil
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
