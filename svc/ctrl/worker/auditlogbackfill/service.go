package auditlogbackfill

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

// Service backfills the legacy audit_log + audit_log_target tables into
// ClickHouse audit_logs_raw_v1. Runs alongside the live drainer; the live
// drainer reads new events from clickhouse_outbox while this service chips
// through the historical tail. Singleton VO keyed "default".
type Service struct {
	hydrav1.UnimplementedAuditLogBackfillServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
	// retentionCh caches the audit log retention window per workspace.
	// Same shape as auditlogexport.Service.retentionCh; backfill stamps
	// expires_at the same way the live drainer does so old rows past the
	// workspace's current retention window get archived on the first cron
	// pass after backfill.
	retentionCh cache.Cache[string, int64]
}

var _ hydrav1.AuditLogBackfillServiceServer = (*Service)(nil)

// Config holds the service dependencies.
type Config struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
	// Heartbeat pings an external monitor after each successful backfill
	// pass. Must not be nil. Use healthcheck.NewNoop() if not needed.
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

	retentionCh, err := cache.New(cache.Config[string, int64]{
		Fresh:    10 * time.Minute,
		Stale:    1 * time.Hour,
		MaxSize:  10000,
		Resource: "audit_log_backfill_retention",
		Clock:    clock.New(),
	})
	if err != nil {
		return nil, fmt.Errorf("create retention cache: %w", err)
	}

	return &Service{
		UnimplementedAuditLogBackfillServiceServer: hydrav1.UnimplementedAuditLogBackfillServiceServer{},
		db:          cfg.DB,
		clickhouse:  cfg.Clickhouse,
		heartbeat:   cfg.Heartbeat,
		retentionCh: retentionCh,
	}, nil
}
