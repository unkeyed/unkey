package auditlogexport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// batchLimit caps the number of outbox rows read per batch. Each row maps to
// one CH row (targets are stored as Nested arrays inside the same row), so
// this also bounds the CH insert size.
const batchLimit int32 = 1000

// knownVersions are the clickhouse_outbox payload versions this drainer
// understands. The SELECT filters on this set so unknown versions stay in
// the table (and are visible via `SELECT version, COUNT(*) FROM
// clickhouse_outbox GROUP BY version` for ops). To roll out a new payload
// shape: deploy a drainer with the new version added to this list FIRST,
// then deploy a writer that emits it.
var knownVersions = []string{auditlog.OutboxVersionV1}

// freeTierRetentionMillis is the retention applied when a workspace has no
// quota row or its `audit_logs_retention_days` is zero. Mirrors the
// freeTierQuotas.auditLogsRetentionDays constant in
// web/apps/dashboard/lib/quotas.ts so the dashboard's retention cutoff and
// the CH TTL stamp agree. Stored as milliseconds so it drops directly into
// the millisecond-based event Time arithmetic when we compute expires_at.
const freeTierRetentionMillis int64 = 30 * 24 * 60 * 60 * 1000

// batchResult is the journaled outcome of a single outbox -> CH batch.
type batchResult struct {
	EventsExported int32 `json:"events_exported"`
}

// RunExport drains the clickhouse_outbox table into ClickHouse in batches.
// Each batch is its own restate.Run so a crash mid-drain only replays the
// last incomplete batch. Within a batch:
//
//  1. SELECT outbox rows WHERE deleted_at IS NULL ORDER BY pk LIMIT batchLimit FOR UPDATE SKIP LOCKED
//  2. Decode the JSON payload into auditlog.Event
//  3. Map to schema.AuditLogV1 (one CH row per event, Nested targets)
//  4. Insert into ClickHouse
//  5. UPDATE deleted_at on the outbox rows (soft delete)
//
// CH insert before mark means a crash after (4) but before (5) leaves the
// outbox rows with deleted_at IS NULL; the next run re-inserts the same
// set in the same order, and CH's block deduplication window collapses the
// duplicate write into a noop. Marked rows stay in the table for ops to
// re-queue (clear deleted_at) and as an audit trail of what was exported.
func (s *Service) RunExport(
	ctx restate.ObjectContext,
	_ *hydrav1.RunExportRequest,
) (*hydrav1.RunExportResponse, error) {
	logger.Info("running audit log export")
	start := time.Now()

	var totalExported int32
	for batchNum := 0; ; batchNum++ {
		result, err := restate.Run(ctx, func(rc restate.RunContext) (batchResult, error) {
			return s.exportBatch(rc)
		}, restate.WithName(fmt.Sprintf("batch-%d", batchNum)))
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", batchNum, err)
		}

		totalExported += result.EventsExported

		if result.EventsExported < batchLimit {
			break
		}
	}

	logger.Info("audit log export complete",
		"events_exported", totalExported,
		"elapsed", time.Since(start),
	)

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunExportResponse{
		EventsExported: totalExported,
	}, nil
}

// exportBatch reads one batch of outbox rows, writes them to ClickHouse,
// then marks them deleted. The whole batch runs inside a single MySQL
// transaction so SELECT FOR UPDATE SKIP LOCKED and the UPDATE land
// atomically. Returns 0 when the outbox is empty.
//
// Failure modes:
//   - JSON decode fails on a row: batch fails, the bad row blocks all
//     progress until investigated. Considered acceptable: malformed
//     payloads are a writer bug, not transient.
//   - CH insert fails: tx rolls back, row locks released, rows stay
//     unmarked, next cron tick retries.
//   - MySQL commit fails after a successful CH insert: rows stay
//     unmarked, next cron re-reads the same set in the same order, re-
//     inserts an identical block, CH's non_replicated_deduplication_window
//     collapses it to a noop, then commits.
func (s *Service) exportBatch(ctx context.Context) (batchResult, error) {
	return db.TxWithResult(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (batchResult, error) {
		rows, err := db.Query.FindClickhouseOutboxBatch(txCtx, tx, db.FindClickhouseOutboxBatchParams{
			Versions: knownVersions,
			Limit:    batchLimit,
		})
		if err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("find outbox batch: %w", err)
		}
		if len(rows) == 0 {
			return batchResult{EventsExported: 0}, nil
		}

		events := make([]auditlog.Event, len(rows))
		pks := make([]uint64, len(rows))
		for i, row := range rows {
			if err := json.Unmarshal(row.Payload, &events[i]); err != nil {
				return batchResult{EventsExported: 0}, fmt.Errorf("decode outbox payload pk=%d: %w", row.Pk, err)
			}
			pks[i] = row.Pk
		}

		retentionByWorkspace, err := s.loadRetentionMillis(txCtx, events)
		if err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("load retention: %w", err)
		}

		chRows, err := buildCHRows(events, retentionByWorkspace)
		if err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("build clickhouse rows: %w", err)
		}

		if err := s.clickhouse.InsertAuditLogs(txCtx, chRows); err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("insert clickhouse: %w", err)
		}

		if err := db.Query.MarkClickhouseOutboxBatchDeleted(txCtx, tx, db.MarkClickhouseOutboxBatchDeletedParams{
			DeletedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			Pks:       pks,
		}); err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("mark outbox batch deleted: %w", err)
		}

		return batchResult{EventsExported: int32(len(events))}, nil
	})
}

// buildCHRows maps decoded outbox events to ClickHouse rows. Targets are
// fanned into parallel arrays so a single event lands as a single CH row
// with Nested target columns; this avoids the GROUP BY reconstruction the
// previous (one-row-per-target) layout required on read.
func buildCHRows(
	events []auditlog.Event,
	retentionByWorkspace map[string]int64,
) ([]schema.AuditLogV1, error) {
	nowMillis := time.Now().UnixMilli()
	out := make([]schema.AuditLogV1, len(events))
	for i, e := range events {
		retentionMs := retentionByWorkspace[e.WorkspaceID]

		actorMeta, err := marshalMeta(e.Actor.Meta)
		if err != nil {
			return nil, fmt.Errorf("encode actor_meta event_id=%s: %w", e.EventID, err)
		}
		topMeta, err := marshalMeta(e.Meta)
		if err != nil {
			return nil, fmt.Errorf("encode meta event_id=%s: %w", e.EventID, err)
		}

		targetTypes := make([]string, len(e.Targets))
		targetIDs := make([]string, len(e.Targets))
		targetNames := make([]string, len(e.Targets))
		targetMetas := make([]json.RawMessage, len(e.Targets))
		for j, t := range e.Targets {
			targetTypes[j] = t.Type
			targetIDs[j] = t.ID
			targetNames[j] = t.Name
			tm, err := marshalMeta(t.Meta)
			if err != nil {
				return nil, fmt.Errorf("encode target_meta event_id=%s target_id=%s: %w", e.EventID, t.ID, err)
			}
			targetMetas[j] = tm
		}

		source := e.Source
		if source == "" {
			source = auditlog.EventSourcePlatform
		}

		out[i] = schema.AuditLogV1{
			EventID:     e.EventID,
			Time:        e.Time,
			InsertedAt:  nowMillis,
			WorkspaceID: e.WorkspaceID,
			Bucket:      e.Bucket,
			Source:      source,
			Event:       e.Event,
			Description: e.Description,
			ActorType:   e.Actor.Type,
			ActorID:     e.Actor.ID,
			ActorName:   e.Actor.Name,
			ActorMeta:   actorMeta,
			RemoteIP:    e.RemoteIP,
			UserAgent:   e.UserAgent,
			Meta:        topMeta,
			TargetTypes: targetTypes,
			TargetIDs:   targetIDs,
			TargetNames: targetNames,
			TargetMetas: targetMetas,
			ExpiresAt:   e.Time + retentionMs,
		}
	}
	return out, nil
}

// marshalMeta returns a JSON object for the CH JSON column. Empty/nil maps
// collapse to {} so the column always holds a parseable JSON value (the
// CH JSON type rejects raw nulls in some configurations).
func marshalMeta(m map[string]any) (json.RawMessage, error) {
	if len(m) == 0 {
		return json.RawMessage("{}"), nil
	}
	return json.Marshal(m)
}

// loadRetentionMillis returns a map from workspace ID to the retention
// window (in milliseconds) that should apply to the batch's events. Results
// are cached via s.retentionCh (10m fresh, 1h stale), so a steady-state
// export drains with zero extra MySQL reads once quotas are warm.
func (s *Service) loadRetentionMillis(
	ctx context.Context,
	events []auditlog.Event,
) (map[string]int64, error) {
	workspaces := make(map[string]struct{}, len(events))
	for _, e := range events {
		workspaces[e.WorkspaceID] = struct{}{}
	}

	out := make(map[string]int64, len(workspaces))
	for ws := range workspaces {
		ms, _, err := s.retentionCh.SWR(ctx, ws,
			func(ctx context.Context) (int64, error) {
				return s.fetchRetentionMillis(ctx, ws)
			},
			func(err error) cache.Op {
				if err == nil {
					return cache.WriteValue
				}
				return cache.Noop
			},
		)
		if err != nil {
			return nil, fmt.Errorf("lookup quota %q: %w", ws, err)
		}
		out[ws] = ms
	}
	return out, nil
}

// fetchRetentionMillis reads audit_logs_retention_days for one workspace
// from MySQL and converts to milliseconds. Missing quota rows or zero values
// collapse to freeTierRetentionMillis so the caller never has to distinguish
// "no quota" from "free tier."
func (s *Service) fetchRetentionMillis(ctx context.Context, workspaceID string) (int64, error) {
	quota, err := db.Query.FindQuotaByWorkspaceID(ctx, s.db.RO(), workspaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return freeTierRetentionMillis, nil
	}
	if err != nil {
		return 0, err
	}
	if quota.AuditLogsRetentionDays <= 0 {
		return freeTierRetentionMillis, nil
	}
	return int64(quota.AuditLogsRetentionDays) * 24 * 60 * 60 * 1000, nil
}
