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

// WorkspaceRateLimitRequest contains everything needed to enforce and record
// a workspace-level rate limit check.
type WorkspaceRateLimitRequest struct {
	// Session is the current request session for analytics. May be nil in tests.
	Session *zen.Session

	// AuthorizedWorkspaceID is the workspace being accessed (ForWorkspaceID).
	// Used for quota lookup and as the ratelimit identifier.
	AuthorizedWorkspaceID string

	// RootKeyWorkspaceID is the workspace the root key lives in.
	// The ratelimit namespace and analytics are written here so the
	// root key owner sees the logs.
	RootKeyWorkspaceID string

	// Audit context for namespace creation audit logs.
	Audit *namespace.AuditContext
}

// checkWorkspaceRateLimit enforces per-workspace API rate limiting based on
// the quota table, using a real ratelimit namespace for analytics.
//
// NULL limit/duration = unlimited (no rate limiting configured).
// 0 limit = zero requests allowed.
// On any internal error (cache miss, rate limiter failure) the check fails
// open to avoid blocking legitimate traffic.
func (s *service) checkWorkspaceRateLimit(ctx context.Context, req WorkspaceRateLimitRequest) error {

	quota, _, err := s.quotaCache.SWR(ctx, req.AuthorizedWorkspaceID, func(ctx context.Context) (db.Quotum, error) {
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

	// Resolve real namespace in the root key's workspace for analytics
	var namespaceID string

	ns, found, nsErr := s.ratelimitNamespaceService.Get(ctx, req.RootKeyWorkspaceID, workspaceRatelimitNamespace)
	if nsErr != nil {
		logger.Warn("workspace rate limit: failed to get namespace",
			"workspace_id", req.RootKeyWorkspaceID,
			"error", nsErr.Error(),
		)
	} else if !found {
		ns, nsErr = s.ratelimitNamespaceService.Create(ctx, req.RootKeyWorkspaceID, workspaceRatelimitNamespace, req.Audit)
		if nsErr != nil {
			logger.Warn("workspace rate limit: failed to create namespace",
				"workspace_id", req.RootKeyWorkspaceID,
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

	rlStart := time.Now()
	resp, err := s.rateLimiter.Ratelimit(ctx, ratelimit.RatelimitRequest{
		Name:       rlName,
		Identifier: req.AuthorizedWorkspaceID,
		Limit:      int64(quota.RatelimitLimit.Int32),
		Duration:   time.Duration(quota.RatelimitDuration.Int32) * time.Millisecond,
		Cost:       1,
		Time:       time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	rlLatency := time.Since(rlStart)
	if err != nil {
		logger.Warn("workspace rate limit: ratelimiter error",
			"workspace_id", req.AuthorizedWorkspaceID,
			"error", err.Error(),
		)
		return nil // fail open
	}

	// Emit analytics in the root key's workspace so the owner sees the logs
	if namespaceID != "" && req.Session != nil {
		s.clickhouse.BufferRatelimit(schema.Ratelimit{
			RequestID:   req.Session.RequestID(),
			WorkspaceID: req.RootKeyWorkspaceID,
			Time:        time.Now().UnixMilli(),
			NamespaceID: namespaceID,
			Identifier:  req.AuthorizedWorkspaceID,
			Passed:      resp.Success,
			Latency:     float64(rlLatency.Milliseconds()),
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
