package keys

import (
	"context"
	"fmt"
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

// checkWorkspaceRateLimit enforces per-workspace API rate limiting based on
// the quota table.
//
// NULL limit/duration = unlimited (no rate limiting configured).
// 0 limit = zero requests allowed.
// On any internal error (cache miss, rate limiter failure) the check fails
// open to avoid blocking legitimate traffic.
func (s *service) checkWorkspaceRateLimit(ctx context.Context, sess *zen.Session) error {
	// When quotaCache is nil, workspace rate limiting is disabled (e.g. sentinel).
	if s.quotaCache == nil {
		return nil
	}

	quota, _, err := s.quotaCache.SWR(ctx, sess.AuthorizedWorkspaceID(), func(ctx context.Context) (db.Quotas, error) {
		return db.Query.FindQuotaByWorkspaceID(ctx, s.db.RO(), sess.AuthorizedWorkspaceID())
	}, caches.DefaultFindFirstOp)
	if err != nil {
		logger.Error("workspace rate limit: failed to load quota",
			"workspace_id", sess.AuthorizedWorkspaceID(),
			"error", err.Error(),
		)
		return nil // fail open
	}

	// NULL = unlimited, no rate limiting configured
	if !quota.RatelimitApiLimit.Valid || !quota.RatelimitApiDuration.Valid {
		return nil
	}

	limit := quota.RatelimitApiLimit.Int32
	duration := time.Duration(quota.RatelimitApiDuration.Int32) * time.Millisecond

	// 0 = explicitly blocked, no requests allowed
	if limit == 0 || duration == 0 {
		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit is zero"),
			fault.Public(
				fmt.Sprintf("This workspace has exceeded its API rate limit of %d/%s. Please try again later.", limit, duration.String()),
			),
		)
	}

	resp, err := s.rateLimiter.Ratelimit(ctx, ratelimit.RatelimitRequest{
		Name:       workspaceRatelimitNamespace,
		Identifier: sess.AuthorizedWorkspaceID(),
		Limit:      int64(limit),
		Duration:   duration,
		Cost:       1,
		Time:       time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		logger.Error("workspace rate limit: ratelimiter error",
			"workspace_id", sess.AuthorizedWorkspaceID(),
			"error", err.Error(),
		)
		return nil // fail open
	}

	// Set standard rate limit headers (IETF draft-ietf-httpapi-ratelimit-headers)
	resetSeconds := max(int64(time.Until(resp.Reset).Seconds()), 0)

	sess.AddHeader("RateLimit-Limit", strconv.FormatInt(resp.Limit, 10))
	sess.AddHeader("RateLimit-Remaining", strconv.FormatInt(resp.Remaining, 10))
	sess.AddHeader("RateLimit-Reset", strconv.FormatInt(resetSeconds, 10))

	if !resp.Success {
		sess.AddHeader("Retry-After", strconv.FormatInt(resetSeconds, 10))

		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit exceeded"),
			fault.Public(
				fmt.Sprintf("This workspace has exceeded its API rate limit of %d/%s. Please try again later.", limit, duration.String()),
			),
		)
	}

	return nil
}
