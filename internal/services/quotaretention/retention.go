// Package quotaretention resolves the operational logs retention window
// for a workspace and exposes it as a single function the analytics
// writers can call on the hot path.
//
// The writers (key verifier, ratelimit handlers, API request middleware,
// sentinel middleware) need a millisecond TTL stamp at insert time. They
// don't have direct access to MySQL or to the workspace quota cache; the
// API service wires both at startup and passes the resolver into each
// writer so the hot path stays simple.
package quotaretention

import (
	"context"
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// FreeTierLogsRetentionDays applies when the workspace has no quota row
// or its logs_retention_days is zero. Mirrors the dashboard's free-tier
// constant; keep in sync with web/apps/dashboard/lib/quotas.ts.
const FreeTierLogsRetentionDays = 30

// dayMillis converts retention days to milliseconds for stamping into
// the analytics tables' Int64 expires_at column.
const dayMillis int64 = 24 * 60 * 60 * 1000

// LogsRetentionMillisFn returns the retention window for one workspace
// in milliseconds. Failures, missing rows, and zero values all collapse
// to the free tier default so callers never have to distinguish them.
type LogsRetentionMillisFn func(ctx context.Context, workspaceID string) int64

// New builds a resolver that reads from the SWR-backed workspace quota
// cache (cache hit on the hot path; cold lookup falls through to MySQL
// via the provided database).
//
// Pass nil for either dependency to get a resolver that always returns
// the free-tier default — used by services that don't have MySQL access
// (e.g. sentinel) but still call the resolver for shape uniformity.
func New(
	quotaCache cache.Cache[string, keysdb.Quotas],
	database db.Database,
) LogsRetentionMillisFn {
	if quotaCache == nil || database == nil {
		return func(context.Context, string) int64 {
			return int64(FreeTierLogsRetentionDays) * dayMillis
		}
	}
	return func(ctx context.Context, workspaceID string) int64 {
		quota, _, err := quotaCache.SWR(ctx, workspaceID, func(ctx context.Context) (keysdb.Quotas, error) {
			return keysdb.Query.FindQuotaByWorkspaceID(ctx, database.RO(), workspaceID)
		}, caches.DefaultFindFirstOp)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			logger.Error("logs retention lookup failed",
				"workspace_id", workspaceID,
				"error", err.Error(),
			)
			return int64(FreeTierLogsRetentionDays) * dayMillis
		}
		if quota.LogsRetentionDays <= 0 {
			return int64(FreeTierLogsRetentionDays) * dayMillis
		}
		return int64(quota.LogsRetentionDays) * dayMillis
	}
}
