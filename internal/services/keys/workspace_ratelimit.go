package keys

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
)

const workspaceRatelimitNamespace = "workspace.ratelimit"

// checkWorkspaceRateLimit enforces per-workspace API rate limiting based on
// the quota table, using a real ratelimit namespace for analytics.
// NULL limit/duration = unlimited (no rate limiting configured).
// 0 limit = zero requests allowed.
// On any internal error (cache miss, rate limiter failure) the check fails
// open to avoid blocking legitimate traffic.
func (s *service) checkWorkspaceRateLimit(ctx context.Context, sess *zen.Session, workspaceID string, audit *namespace.AuditContext) error {

	quota, _, err := s.quotaCache.SWR(ctx, workspaceID, func(ctx context.Context) (db.Quotum, error) {
		return db.Query.FindQuotaByWorkspaceID(ctx, s.db.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		logger.Warn("workspace rate limit: failed to load quota",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	// NULL = unlimited, no rate limiting configured
	if !quota.RatelimitLimit.Valid || !quota.RatelimitDuration.Valid {
		return nil
	}

	// Resolve real namespace for analytics — fail open if unavailable
	var namespaceID string

	ns, found, nsErr := s.ratelimitNamespaceService.Get(ctx, workspaceID, workspaceRatelimitNamespace)
	if nsErr != nil {
		logger.Warn("workspace rate limit: failed to get namespace",
			"workspace_id", workspaceID,
			"error", nsErr.Error(),
		)
	} else if !found {
		ns, nsErr = s.ratelimitNamespaceService.Create(ctx, workspaceID, workspaceRatelimitNamespace, audit)
		if nsErr != nil {
			logger.Warn("workspace rate limit: failed to create namespace",
				"workspace_id", workspaceID,
				"error", nsErr.Error(),
			)
		} else {
			namespaceID = ns.ID
		}
	} else {
		namespaceID = ns.ID
	}

	// Use namespace ID as the ratelimit name when available, otherwise fall back
	rlName := workspaceRatelimitNamespace
	if namespaceID != "" {
		rlName = namespaceID
	}

	resp, err := s.rateLimiter.Ratelimit(ctx, ratelimit.RatelimitRequest{
		Name:       rlName,
		Identifier: workspaceID,
		Limit:      int64(quota.RatelimitLimit.Int32),
		Duration:   time.Duration(quota.RatelimitDuration.Int32) * time.Millisecond,
		Cost:       1,
		Time:       time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		logger.Warn("workspace rate limit: ratelimiter error",
			"workspace_id", workspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	// Emit analytics with real namespace ID when available
	if namespaceID != "" && sess != nil {
		s.clickhouse.BufferRatelimit(schema.Ratelimit{
			RequestID:   sess.RequestID(),
			WorkspaceID: workspaceID,
			Time:        time.Now().UnixMilli(),
			NamespaceID: namespaceID,
			Identifier:  workspaceID,
			Passed:      resp.Success,
			Latency:     0,
			OverrideID:  "",
			Limit:       uint64(resp.Limit),
			Remaining:   uint64(resp.Remaining),
			ResetAt:     resp.Reset.UnixMilli(),
		})
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
