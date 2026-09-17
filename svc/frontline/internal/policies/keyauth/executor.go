package keyauth

import (
	"context"
	"net/http"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/principal"
)

// Executor handles credential extraction and KeyAuth response semantics.
type Executor struct {
	verifier Verifier
	clock    clock.Clock
}

// New creates a KeyAuth policy executor with the selected verification backend.
func New(verifier Verifier, clk clock.Clock) *Executor {
	return &Executor{
		verifier: verifier,
		clock:    clk,
	}
}

// Execute evaluates a KeyAuth policy against the incoming request.
// It extracts the API key and returns a Principal on successful verification.
func (e *Executor) Execute(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	appID string,
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

	result, err := e.verifier.Verify(ctx, sess, VerifyRequest{
		RawKey:          rawKey,
		AppID:           appID,
		Keyspaces:       cfg.GetKeySpaceIds(),
		Credits:         ptr.SafeDeref(cfg.Credits, 1),
		PermissionQuery: cfg.GetPermissionQuery(),
		Ratelimits:      toVerifyRatelimits(cfg.GetRatelimits()),
	})
	writeRateLimitHeaders(sess.ResponseWriter(), result.Ratelimits, e.clock)
	if err != nil {
		return nil, err
	}

	switch result.Status {
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
			fault.Code(codes.Frontline.Auth.UsageExceeded.URN()),
			fault.Internal("usage limit exceeded"),
			fault.Public("Usage limit exceeded. This API key has no remaining credits."),
		)
	case keys.StatusNotFound, keys.StatusDisabled, keys.StatusExpired,
		keys.StatusForbidden, keys.StatusWorkspaceDisabled, keys.StatusWorkspaceNotFound:
		return nil, fault.New("key verification failed",
			fault.Code(codes.Frontline.Auth.InvalidKey.URN()),
			fault.Internal("post-verification status: "+string(result.Status)),
			fault.Public("Authentication failed."),
		)
	}

	if err := assert.All(
		assert.Equal(result.Status, keys.StatusValid, "unexpected verification status"),
		assert.NotNilAndNotZero(result.Principal, "successful verification requires a principal"),
	); err != nil {
		return nil, fault.Wrap(err,
			fault.Code(codes.Frontline.Internal.InternalServerError.URN()),
			fault.Public("An internal error occurred during authentication."),
		)
	}
	return result.Principal, nil
}

// toVerifyRatelimits converts the policy's rate limit selectors into the
// verifyKey-style options consumed by KeyService. This reuses the exact
// enforcement path the API uses for verifyKey's ratelimits parameter, so
// gateway-enforced limits behave identically to application-enforced ones.
func toVerifyRatelimits(rls []*frontlinev1.KeyRatelimit) []openapi.KeysVerifyKeyRatelimit {
	out := make([]openapi.KeysVerifyKeyRatelimit, 0, len(rls))
	for _, rl := range rls {
		entry := openapi.KeysVerifyKeyRatelimit{
			Name:     rl.GetName(),
			Cost:     nil,
			Duration: nil,
			Limit:    nil,
		}
		if rl.Limit != nil {
			entry.Limit = ptr.P(int(rl.GetLimit()))
		}
		if rl.Duration != nil {
			entry.Duration = ptr.P(int(rl.GetDuration()))
		}
		if rl.Cost != nil {
			entry.Cost = ptr.P(int(rl.GetCost()))
		}
		out = append(out, entry)
	}
	return out
}
