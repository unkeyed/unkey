package keys

import (
	"context"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

const workspaceRatelimitNamespace = "workspace.ratelimit"

// WorkspaceRateLimitRequest contains everything needed to enforce
// a workspace-level rate limit check.
type WorkspaceRateLimitRequest struct {
	// Session is the current request session. May be nil in tests.
	Session *zen.Session

	// AuthorizedWorkspaceID is the workspace being accessed (ForWorkspaceID).
	// Used for quota lookup and as the ratelimit identifier.
	AuthorizedWorkspaceID string
}

// checkWorkspaceRateLimit enforces per-workspace API rate limiting based on
// the quota table.
//
// NULL limit/duration = unlimited (no rate limiting configured).
// 0 limit = zero requests allowed.
// On any internal error (cache miss, rate limiter failure) the check fails
// open to avoid blocking legitimate traffic.
func (s *service) checkWorkspaceRateLimit(ctx context.Context, req WorkspaceRateLimitRequest) error {

	quota, _, err := s.quotaCache.SWR(ctx, req.AuthorizedWorkspaceID, func(ctx context.Context) (db.Quotas, error) {
		return db.Query.FindQuotaByWorkspaceID(ctx, s.db.RO(), req.AuthorizedWorkspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		logger.Warn("workspace rate limit: failed to load quota",
			"workspace_id", req.AuthorizedWorkspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	// NULL = unlimited, no rate limiting configured
	if !quota.RatelimitLimit.Valid || !quota.RatelimitDuration.Valid {
		return nil
	}

	limit := quota.RatelimitLimit.Int32
	duration := time.Duration(quota.RatelimitDuration.Int32) * time.Millisecond

	// 0 = explicitly blocked, no requests allowed
	if limit == 0 || duration == 0 {
		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit is zero"),
			fault.Public(
				"This workspace has exceeded its API rate limit of "+strconv.Itoa(int(limit))+"/"+formatDuration(duration)+". Please try again later.",
			),
		)
	}

	resp, err := s.rateLimiter.Ratelimit(ctx, ratelimit.RatelimitRequest{
		Name:       workspaceRatelimitNamespace,
		Identifier: req.AuthorizedWorkspaceID,
		Limit:      int64(limit),
		Duration:   duration,
		Cost:       1,
		Time:       time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		logger.Warn("workspace rate limit: ratelimiter error",
			"workspace_id", req.AuthorizedWorkspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	// Set standard rate limit headers (IETF draft-ietf-httpapi-ratelimit-headers)
	if req.Session != nil {
		resetSeconds := max(int64(time.Until(resp.Reset).Seconds()), 0)

		req.Session.AddHeader("RateLimit-Limit", strconv.FormatInt(resp.Limit, 10))
		req.Session.AddHeader("RateLimit-Remaining", strconv.FormatInt(resp.Remaining, 10))
		req.Session.AddHeader("RateLimit-Reset", strconv.FormatInt(resetSeconds, 10))

		if !resp.Success {
			req.Session.AddHeader("Retry-After", strconv.FormatInt(resetSeconds, 10))
		}
	}

	if !resp.Success {
		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit exceeded"),
			fault.Public(
				"This workspace has exceeded its API rate limit of "+strconv.Itoa(int(limit))+"/"+formatDuration(duration)+". Please try again later.",
			),
		)
	}

	return nil
}

// formatDuration returns a compact human-readable duration like "2s", "500ms", "1m0s".
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return strconv.FormatInt(d.Milliseconds(), 10) + "ms"
	}
	return d.Truncate(time.Second).String()
}
