package keys

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// checkWorkspaceRateLimit enforces per-workspace API rate limiting based on
// the quota table. If the workspace has no quota row, or the rate limit is
// configured as 0 (unlimited), the request is allowed. On any internal error
// (cache miss, rate limiter failure) the check fails open to avoid blocking
// legitimate traffic.
func (s *service) checkWorkspaceRateLimit(ctx context.Context, workspaceID string) error {
	if s.quotaCache == nil {
		return nil
	}

	quota, hit, err := s.quotaCache.SWR(ctx, workspaceID, func(ctx context.Context) (db.Quotum, error) {
		return db.Query.FindQuotaByWorkspaceID(ctx, s.db.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		logger.Warn("workspace rate limit: failed to load quota",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	if hit == cache.Null {
		return nil // no quota row — unlimited
	}

	if quota.RatelimitLimit <= 0 || quota.RatelimitDuration <= 0 {
		return nil // rate limiting not configured
	}

	resp, err := s.rateLimiter.Ratelimit(ctx, ratelimit.RatelimitRequest{
		Name:       "__unkey_workspace_api_ratelimit",
		Identifier: workspaceID,
		Limit:      quota.RatelimitLimit,
		Duration:   time.Duration(quota.RatelimitDuration) * time.Millisecond,
		Cost:       1,
	})
	if err != nil {
		logger.Warn("workspace rate limit: ratelimiter error",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	if !resp.Success {
		return fault.New("workspace rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.WorkspaceRateLimited.URN()),
			fault.Internal("workspace rate limit exceeded"),
			fault.Public("This workspace has exceeded its API rate limit. Please try again later."),
		)
	}

	return nil
}
