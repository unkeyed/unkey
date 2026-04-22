package auditlogexport

import (
	"fmt"
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

// Service ships rows from the MySQL `audit_log` outbox into the ClickHouse
// `audit_logs_raw_v1` table. Registered as a Restate virtual object keyed by
// "default" so concurrent cron triggers queue instead of racing.
type Service struct {
	hydrav1.UnimplementedAuditLogExportServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
	// retentionCh caches the audit log retention window per workspace.
	// Key: workspace_id (e.g. "ws_abc123").
	// Value: retention in milliseconds, post-fallback. Stored in ms rather
	// than days so it drops straight into time.UnixMilli arithmetic when we
	// compute each row's expires_at. A hit of 2592000000 means "30 day
	// retention applies to this workspace's audit logs", regardless of
	// whether it came from a real quota row or the freeTierRetentionMillis
	// default. Consumers don't need to distinguish the two.
	retentionCh cache.Cache[string, int64]
}

var _ hydrav1.AuditLogExportServiceServer = (*Service)(nil)

// Config holds the service dependencies.
type Config struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
	// Heartbeat pings an external monitor after each successful drain.
	// Must not be nil. Use healthcheck.NewNoop() if not needed.
	Heartbeat healthcheck.Heartbeat
}

// New constructs the service.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop() if not needed"),
	); err != nil {
		return nil, err
	}

	// Retention changes only when a workspace's plan changes (rare and
	// never time-critical), since a few minutes of staleness only shifts
	// the TTL stamp a few minutes (rows expire days later either way).
	// Fresh 10m means most batches pay no MySQL hit; stale 1h keeps the
	// value available through transient MySQL blips via SWR background
	// refresh.
	retentionCh, err := cache.New(cache.Config[string, int64]{
		Fresh:    10 * time.Minute,
		Stale:    1 * time.Hour,
		MaxSize:  10000,
		Resource: "audit_log_export_retention",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("create retention cache: %w", err)
	}

	return &Service{
		UnimplementedAuditLogExportServiceServer: hydrav1.UnimplementedAuditLogExportServiceServer{},
		db:                                       cfg.DB,
		clickhouse:                               cfg.Clickhouse,
		heartbeat:                                cfg.Heartbeat,
		retentionCh:                              retentionCh,
	}, nil
}
