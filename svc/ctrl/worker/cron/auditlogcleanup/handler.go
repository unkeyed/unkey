// Package auditlogcleanup implements the
// CronService.RunAuditLogOutboxCleanup handler. The handler hard-deletes
// already-exported clickhouse_outbox rows (deleted_at stamped) older than
// the retention window so the outbox stays bounded. The audit log export
// drainer (auditlogexport) soft-deletes rows instead of removing them so
// ops can re-queue or audit recently-exported events; this sweep reclaims
// that space once the window has passed.
package auditlogcleanup

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
)

// retention is how long an exported (soft-deleted) outbox row is kept before
// this sweep hard-deletes it. The window leaves headroom for ops to re-queue
// (clear deleted_at) or audit recently-exported events.
const retention = 30 * 24 * time.Hour

// batchLimit bounds each DELETE so row locks stay short and replication lag
// stays bounded; the handler loops until a batch deletes fewer than this.
const batchLimit int32 = 10000

// Config holds the handler's dependencies.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database

	// Heartbeat is pinged after a successful sweep. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunAuditLogOutboxCleanup.
type Handler struct {
	db        db.Database
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, heartbeat: cfg.Heartbeat}, nil
}

// Handle deletes every clickhouse_outbox row whose deleted_at is older than
// the retention cutoff, in bounded batches. Pending rows (deleted_at IS NULL)
// are never matched. Each batch DELETE is wrapped in restate.Run so a crash
// or retry replays cleanly: at-least-once delivery on a deterministic,
// cutoff-bounded DELETE is safe — re-running only removes rows that were
// already eligible.
//
// Stateless — the VO key is fixed at "audit-log-outbox-cleanup" so a
// paused/wedged invocation cannot block other cron handlers.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunAuditLogOutboxCleanupRequest,
) (*hydrav1.RunAuditLogOutboxCleanupResponse, error) {
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get now: %w", err)
	}
	cutoff := now.Add(-retention).UnixMilli()

	var totalDeleted int64
	for batchNum := 0; ; batchNum++ {
		deleted, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return db.Query.DeleteExportedClickhouseOutbox(rc, h.db.RW(), db.DeleteExportedClickhouseOutboxParams{
				Cutoff: sql.NullInt64{Int64: cutoff, Valid: true},
				Limit:  batchLimit,
			})
		}, restate.WithName(fmt.Sprintf("delete batch-%d", batchNum)))
		if err != nil {
			return nil, fmt.Errorf("delete exported outbox batch %d: %w", batchNum, err)
		}

		totalDeleted += deleted

		if deleted < int64(batchLimit) {
			break
		}
	}

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunAuditLogOutboxCleanupResponse{
		RowsDeleted: totalDeleted,
	}, nil
}
