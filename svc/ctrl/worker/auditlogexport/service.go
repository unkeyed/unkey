package auditlogexport

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

// Service ships rows from the MySQL `audit_log` outbox into the ClickHouse
// `audit_logs_raw_v1` table. Registered as a Restate virtual object keyed by
// "default" so concurrent cron triggers queue instead of racing.
//
// Retention is computed inside CH via the `expires_at` MATERIALIZED
// column on `audit_logs_raw_v1` reading from
// `default.workspace_quota_dict`. The dict is mirrored from MySQL by
// the WorkspaceQuotaSyncService VO, so this writer needs no per-
// workspace lookup.
type Service struct {
	hydrav1.UnimplementedAuditLogExportServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
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
	return &Service{
		UnimplementedAuditLogExportServiceServer: hydrav1.UnimplementedAuditLogExportServiceServer{},
		db:                                       cfg.DB,
		clickhouse:                               cfg.Clickhouse,
		heartbeat:                                cfg.Heartbeat,
	}, nil
}
