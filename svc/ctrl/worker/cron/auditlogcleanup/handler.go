// Package auditlogcleanup implements the
// CronService.RunAuditLogOutboxCleanup handler. The handler hard-deletes
// already-exported outbox rows older than the retention window so the
// outboxes stay bounded. It sweeps two tables that follow the same
// mark-then-sweep contract:
//
//   - clickhouse_outbox: the audit log export drainer (auditlogexport)
//     stamps deleted_at instead of removing rows, so ops can re-queue or
//     audit recently-exported events.
//   - backoffice_audit_outbox: the staff back office app stamps drained_at
//     the same way after copying a row to ClickHouse. Its MySQL user has no
//     DELETE grant, so this cron is the only thing that removes its rows.
//
// This sweep reclaims that space once the window has passed.
package auditlogcleanup

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// retention is how long an exported (soft-deleted) outbox row is kept before
// this sweep hard-deletes it. The window leaves headroom for ops to re-queue
// (clear deleted_at / drained_at) or audit recently-exported events. One
// window for both tables: the back office rows are few and follow the same
// contract.
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

// deleteBatch removes one bounded batch of exported rows older than cutoff
// from one outbox table and reports how many it removed.
type deleteBatch func(ctx context.Context, cutoff int64, limit int32) (int64, error)

// Handle deletes every clickhouse_outbox row whose deleted_at, and every
// backoffice_audit_outbox row whose drained_at, is older than the retention
// cutoff, in bounded batches. Pending rows (stamp IS NULL) are never
// matched. Each batch DELETE is wrapped in restate.Run so a crash or retry
// replays cleanly: at-least-once delivery on a deterministic, cutoff-bounded
// DELETE is safe — re-running only removes rows that were already eligible.
//
// The tables are swept one after the other; a failure on the first stops
// the run before the second, and the next daily tick retries both.
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

	sweeps := []struct {
		table  string
		delete deleteBatch
	}{
		{
			table: "clickhouse_outbox",
			delete: func(ctx context.Context, cutoff int64, limit int32) (int64, error) {
				return h.db.DeleteExportedClickhouseOutbox(ctx, db.DeleteExportedClickhouseOutboxParams{
					Cutoff: sql.NullInt64{Int64: cutoff, Valid: true},
					Limit:  limit,
				})
			},
		},
		{
			table: "backoffice_audit_outbox",
			delete: func(ctx context.Context, cutoff int64, limit int32) (int64, error) {
				return h.db.DeleteDrainedBackofficeAuditOutbox(ctx, db.DeleteDrainedBackofficeAuditOutboxParams{
					Cutoff: sql.NullInt64{Int64: cutoff, Valid: true},
					Limit:  limit,
				})
			},
		},
	}

	var totalDeleted int64
	for _, sweep := range sweeps {
		deleted, err := h.sweep(ctx, sweep.table, sweep.delete, cutoff)
		if err != nil {
			return nil, err
		}
		totalDeleted += deleted
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

// sweep loops one table's bounded DELETE until a batch comes back short.
// The restate.Run name carries the table so journal entries of the two
// sweeps never collide on replay.
func (h *Handler) sweep(
	ctx restate.ObjectContext,
	table string,
	del deleteBatch,
	cutoff int64,
) (int64, error) {
	var total int64
	for batchNum := 0; ; batchNum++ {
		deleted, err := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
			return del(rc, cutoff, batchLimit)
		}, restate.WithName(fmt.Sprintf("delete %s batch-%d", table, batchNum)))
		if err != nil {
			return 0, fmt.Errorf("delete exported %s batch %d: %w", table, batchNum, err)
		}

		total += deleted

		if deleted < int64(batchLimit) {
			return total, nil
		}
	}
}
