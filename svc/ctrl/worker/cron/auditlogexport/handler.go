// Package auditlogexport implements the CronService.RunAuditLogExport
// handler. The handler drains the MySQL clickhouse_outbox table into
// ClickHouse audit_logs_raw_v1.
package auditlogexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"go.opentelemetry.io/otel/attribute"
)

// batchLimit caps the number of outbox rows read per batch. Each row
// maps to one CH row (targets are stored as Nested arrays inside the
// same row), so this also bounds the CH insert size.
const batchLimit int32 = 1000

// knownVersions are the clickhouse_outbox payload versions this drainer
// understands. The SELECT filters on this set so unknown versions stay
// in the table (and are visible via
// `SELECT version, COUNT(*) FROM clickhouse_outbox GROUP BY version` for
// ops). To roll out a new payload shape: deploy a drainer with the new
// version added to this list FIRST, then deploy a writer that emits it.
var knownVersions = []string{auditlog.OutboxVersionV1}

// batchResult is the journaled outcome of a single outbox -> CH batch.
type batchResult struct {
	EventsExported int32 `json:"events_exported"`
}

// Config holds the handler's dependencies.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database
	// Clickhouse is the analytics database. Must not be nil — pass
	// clickhouse.NewNoop() if unavailable.
	Clickhouse clickhouse.ClickHouse
	// Heartbeat is pinged on successful completion. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunAuditLogExport.
type Handler struct {
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop() if unavailable"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		db:         cfg.DB,
		clickhouse: cfg.Clickhouse,
		heartbeat:  cfg.Heartbeat,
	}, nil
}

// Handle drains the clickhouse_outbox table into ClickHouse in batches.
// Singleton VO keyed "audit-log-export" so concurrent cron triggers
// queue. Each batch is its own restate.Run so a crash mid-drain only
// replays the last incomplete batch. Within a batch:
//
//  1. SELECT outbox rows WHERE deleted_at IS NULL ORDER BY pk LIMIT N (autocommit)
//  2. Decode the JSON payload into auditlog.Event
//  3. Map to schema.AuditLogV1 (one CH row per event, Nested targets)
//  4. Insert into ClickHouse
//  5. UPDATE deleted_at on exactly the selected pks (autocommit soft delete)
//
// CH insert before mark means a crash after (4) but before (5) leaves
// the outbox rows with deleted_at IS NULL; the next run re-inserts the
// pending rows. Consumers must tolerate duplicates: block deduplication
// has a finite window, and ReplacingMergeTree does not merge across
// insertion-month partitions. Marked rows remain available for re-queue.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunAuditLogExportRequest,
) (*hydrav1.RunAuditLogExportResponse, error) {
	logger.Debug("running audit log export")
	start := time.Now()

	var totalExported int32
	for batchNum := 0; ; batchNum++ {
		result, err := restate.Run(ctx, func(rc restate.RunContext) (batchResult, error) {
			return h.exportBatch(rc)
		}, restate.WithName(fmt.Sprintf("batch-%d", batchNum)))
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", batchNum, err)
		}

		totalExported += result.EventsExported

		if result.EventsExported < batchLimit {
			break
		}
	}

	logger.Debug("audit log export complete",
		"events_exported", totalExported,
		"elapsed", time.Since(start),
	)

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunAuditLogExportResponse{
		EventsExported: totalExported,
	}, nil
}

// exportBatch reads one batch of outbox rows, writes them to ClickHouse,
// then marks exactly those rows deleted. No MySQL connection spans the
// ClickHouse call, so a stalled export cannot gap-lock API outbox inserts.
// CH failures leave the batch pending. A failed mark can have an unknown
// outcome after connection loss; any rows still pending will be replayed.
func (h *Handler) exportBatch(ctx context.Context) (result batchResult, batchErr error) {
	ctx, span := tracing.Start(ctx, "auditlogexport.exportBatch")
	defer span.End()
	defer func() {
		tracing.RecordError(span, batchErr)
	}()
	span.AddEvent("select")
	rows, err := h.db.FindClickhouseOutboxBatch(ctx, db.FindClickhouseOutboxBatchParams{
		Versions: knownVersions,
		Limit:    batchLimit,
	})
	if err != nil {
		return batchResult{EventsExported: 0}, fmt.Errorf("find outbox batch: %w", err)
	}
	span.SetAttributes(attribute.Int("rows", len(rows)))
	if len(rows) == 0 {
		return batchResult{EventsExported: 0}, nil
	}

	span.AddEvent("decode")
	events := make([]auditlog.Event, len(rows))
	pks := make([]uint64, len(rows))
	for i, row := range rows {
		if err := json.Unmarshal(row.Payload, &events[i]); err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("decode outbox payload pk=%d: %w", row.Pk, err)
		}
		pks[i] = row.Pk
	}

	chRows, err := clickhouse.EncodeAuditLogEvents(events)
	if err != nil {
		return batchResult{EventsExported: 0}, fmt.Errorf("encode clickhouse rows: %w", err)
	}

	span.AddEvent("insert")
	if err := h.clickhouse.InsertAuditLogs(ctx, chRows); err != nil {
		return batchResult{EventsExported: 0}, fmt.Errorf("insert clickhouse: %w", err)
	}

	span.AddEvent("mark")
	if err := h.db.MarkClickhouseOutboxBatchDeleted(ctx, db.MarkClickhouseOutboxBatchDeletedParams{
		DeletedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		Pks:       pks,
	}); err != nil {
		return batchResult{EventsExported: 0}, fmt.Errorf("mark outbox batch deleted: %w", err)
	}

	return batchResult{EventsExported: int32(len(events))}, nil
}
