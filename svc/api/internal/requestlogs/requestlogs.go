package requestlogs

import (
	"context"
	"errors"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/unkeyed/unkey/internal/services/caches"
	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
)

// RetentionStart returns the oldest time in unix milliseconds the workspace
// may read, or 0 when its log retention has no limit
func RetentionStart(ctx context.Context, limitsCache cache.Cache[string, keysdb.Limit], database db.Database, clk clock.Clock, workspaceID string) (int64, error) {
	limits, hit, err := limitsCache.SWR(ctx, workspaceID, func(ctx context.Context) (keysdb.Limit, error) {
		return keysdb.Query.FindLimitsByWorkspaceID(ctx, database.RO(), workspaceID)
	}, caches.DefaultFindFirstOp)
	if db.IsNotFound(err) || hit == cache.Null {
		return 0, domaingate.LimitsNotConfigured(workspaceID)
	}
	if err != nil {
		return 0, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load workspace limits"),
			fault.Public("Failed to load the log retention of the workspace. Try again."),
		)
	}

	if limits.LogsRetentionDaysMax == 0 {
		return 0, nil
	}

	retention := time.Duration(limits.LogsRetentionDaysMax) * 24 * time.Hour
	return clk.Now().Add(-retention).UnixMilli(), nil
}

// ReadError turns the error of a request log read into the API error: a read
// limit is a 4xx, a query ClickHouse rejects is a 500, and anything else is
// the log store being unavailable
func ReadError(err error, internal, readLimitPublic string) error {
	// A fresh fault, because the read limit's own public message tells the
	// caller to simplify a query they did not write
	if readLimitCode, isReadLimit := fault.GetCode(err); isReadLimit {
		return fault.New(internal+": read limit",
			fault.Code(readLimitCode),
			fault.Internal(err.Error()),
			fault.Public(readLimitPublic),
		)
	}
	if _, isException := errors.AsType[*ch.Exception](err); isException {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal(internal+": clickhouse rejected the query"),
			fault.Public("An internal error occurred."),
		)
	}

	return fault.Wrap(err,
		fault.Code(codes.Data.Analytics.ConnectionFailed.URN()),
		fault.Internal(internal),
		fault.Public("The log store is not available. Try again."),
	)
}
