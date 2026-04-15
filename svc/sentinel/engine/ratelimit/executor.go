package ratelimit

import (
	"context"
	"net/http"
	"time"

	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	rl "github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/engine/principal"
)

// Executor handles standalone RateLimit policy evaluation. This is distinct
// from KeyAuth's per-key rate limiting: the standalone policy is user-configured
// at the gateway level (per-route, per-IP, per-header, etc).
type Executor struct {
	rateLimiter rl.Service
	clock       clock.Clock
}

// New creates a new RateLimit policy executor.
func New(rateLimiter rl.Service, clk clock.Clock) *Executor {
	return &Executor{
		rateLimiter: rateLimiter,
		clock:       clk,
	}
}

// Execute evaluates a RateLimit policy against the incoming request.
// It extracts the rate limit key, checks the limit, writes response headers,
// and returns an error if the request is rate limited.
func (e *Executor) Execute(
	ctx context.Context,
	sess *zen.Session,
	req *http.Request,
	policyID string,
	cfg *sentinelv1.RateLimit,
	principal *principal.Principal,
) error {
	identifier := extractIdentifier(sess, req, cfg.GetKey(), principal)
	if identifier == "" {
		return fault.New("missing rate limit identifier",
			fault.Code(codes.Sentinel.Auth.RateLimited.URN()),
			fault.Internal("no rate limit identifier could be resolved from request"),
			fault.Public("Rate limit configuration error. Unable to identify the request."),
		)
	}

	resp, err := e.rateLimiter.Ratelimit(ctx, rl.RatelimitRequest{
		Name:       policyID,
		Identifier: identifier,
		Limit:      cfg.GetLimit(),
		Duration:   time.Duration(cfg.GetWindowMs()) * time.Millisecond,
		Cost:       1,
		Time:       time.Time{},
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.Sentinel.Internal.InternalServerError.URN()),
			fault.Internal("rate limit check failed"),
			fault.Public("An internal error occurred. Please try again later."),
		)
	}

	writePolicyRateLimitHeaders(sess.ResponseWriter(), resp, e.clock)

	if !resp.Success {
		return fault.New("rate limited",
			fault.Code(codes.Sentinel.Auth.RateLimited.URN()),
			fault.Internal("policy rate limit exceeded"),
			fault.Public("Rate limit exceeded. Please try again later."),
		)
	}

	return nil
}
