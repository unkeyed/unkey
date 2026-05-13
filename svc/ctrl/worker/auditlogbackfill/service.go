package auditlogbackfill

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

// Service backfills the legacy audit_log + audit_log_target tables into
// ClickHouse audit_logs_raw_v1. Runs alongside the live drainer; the
// live drainer reads new events from clickhouse_outbox while this
// service chips through the historical tail. Singleton VO keyed
// "default".
//
// Retention is computed inside CH via the `expires_at` MATERIALIZED
// column on `audit_logs_raw_v1`, so this service ships rows without a
// per-workspace retention lookup.
type Service struct {
	hydrav1.UnimplementedAuditLogBackfillServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
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
	return &Service{
		UnimplementedAuditLogBackfillServiceServer: hydrav1.UnimplementedAuditLogBackfillServiceServer{},
		db:         cfg.DB,
		clickhouse: cfg.Clickhouse,
		heartbeat:  cfg.Heartbeat,
	}, nil
}
