package auditlogexport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// batchLimit caps the number of audit log events read per batch. A batch with
// N events times M targets each produces up to N*M rows in ClickHouse, so this
// bounds memory and per-batch MySQL tx duration.
const batchLimit int32 = 1000

// freeTierRetentionMillis is the retention applied when a workspace has no
// quota row or its `audit_logs_retention_days` is zero. Mirrors the
// freeTierQuotas.auditLogsRetentionDays constant in
// web/apps/dashboard/lib/quotas.ts so the dashboard's retention cutoff and
// the CH TTL stamp agree. Stored as milliseconds so it drops directly into
// the millisecond-based event.Time arithmetic when we compute expires_at.
const freeTierRetentionMillis int64 = 30 * 24 * 60 * 60 * 1000

// unkeyPlatformWorkspaceID is the row-level owner for every Unkey-emitted
// platform audit event. Empty string is meaningful here: "no customer
// owns this data, it's platform-internal." Once customer-emitted audit
// logs ship, their rows carry a real workspace_id, and platform rows stay
// distinguishable by empty workspace_id. The originating customer
// workspace is preserved in bucket_id via unkeyPlatformBucketID.
//
// This contract is mirrored by the dashboard reader, see
// web/apps/dashboard/lib/trpc/routers/audit/fetch.ts
// (UNKEY_PLATFORM_WORKSPACE_ID and platformBucketFor). The two MUST stay
// in sync; changing one without the other silently breaks the dashboard.
const unkeyPlatformWorkspaceID = ""

// unkeyPlatformBucketID returns the CH bucket_id for a platform audit event
// emitted about a given customer workspace. The `unkey_audit_` prefix
// visually separates these from customer-defined bucket names so the two
// namespaces can coexist in the same column without collisions once
// customers start emitting their own events.
func unkeyPlatformBucketID(customerWorkspaceID string) string {
	return "unkey_audit_" + customerWorkspaceID
}

// batchResult is the journaled outcome of a single MySQL to ClickHouse batch.
type batchResult struct {
	EventsExported int32 `json:"events_exported"`
}

// RunExport drains the MySQL audit_log outbox into ClickHouse in batches. Each
// batch is its own restate.Run so a crash mid-drain only replays the last
// incomplete batch. Within a batch:
//
//  1. SELECT unexported audit_log rows (ordered by pk, capped at batchLimit)
//  2. Fetch targets for those events
//  3. Fan out to ClickHouse rows (one per target, shared event_id) and insert
//  4. UPDATE audit_log SET exported = true WHERE pk IN (...)
//
// CH insert before MySQL update means a crash after (3) but before (4) leaves
// MySQL rows still marked unexported; the next run re-inserts, and CH's block
// deduplication window collapses the identical re-insert into a noop.
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

// exportBatch reads one batch of unexported audit logs, writes them to
// ClickHouse, and marks them exported. The whole batch runs inside a single
// MySQL transaction so the SELECT FOR UPDATE SKIP LOCKED and the UPDATE land
// atomically. Returns 0 when the outbox is empty.
//
// Failure modes:
//   - CH insert fails: tx rolls back, row locks released, rows stay
//     unexported, next cron tick retries.
//   - MySQL commit fails after a successful CH insert: rows stay unexported,
//     next cron re-reads the same set in the same order, re-inserts an
//     identical block, CH's non_replicated_deduplication_window collapses it
//     to a noop, then commits.
func (s *Service) exportBatch(ctx context.Context) (batchResult, error) {
	return db.TxWithResult(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) (batchResult, error) {
		events, err := db.Query.FindUnexportedAuditLogs(txCtx, tx, batchLimit)
		if err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("find unexported: %w", err)
		}
		if len(events) == 0 {
			return batchResult{EventsExported: 0}, nil
		}

		eventIDs := make([]string, len(events))
		pks := make([]uint64, len(events))
		for i, e := range events {
			eventIDs[i] = e.ID
			pks[i] = e.Pk
		}

		targets, err := db.Query.FindAuditLogTargetsForLogs(txCtx, tx, eventIDs)
		if err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("find targets: %w", err)
		}

		targetsByLog := make(map[string][]db.FindAuditLogTargetsForLogsRow, len(events))
		for _, t := range targets {
			targetsByLog[t.AuditLogID] = append(targetsByLog[t.AuditLogID], t)
		}

		retentionByWorkspace, err := s.loadRetentionMillis(txCtx, events)
		if err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("load retention: %w", err)
		}

		rows := buildCHRows(events, targetsByLog, retentionByWorkspace)

		if err := s.clickhouse.InsertAuditLogs(txCtx, rows); err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("insert clickhouse: %w", err)
		}

		if err := db.Query.MarkAuditLogsExported(txCtx, tx, pks); err != nil {
			return batchResult{EventsExported: 0}, fmt.Errorf("mark exported: %w", err)
		}

		return batchResult{EventsExported: int32(len(events))}, nil
	})
}

// buildCHRows fans out one CH row per (event × target). Events with no
// targets emit a single row with empty target fields so they still appear in
// ClickHouse rather than being silently dropped.
func buildCHRows(
	events []db.FindUnexportedAuditLogsRow,
	targetsByLog map[string][]db.FindAuditLogTargetsForLogsRow,
	retentionByWorkspace map[string]int64,
) []schema.AuditLogV1 {
	out := make([]schema.AuditLogV1, 0, len(events))
	for _, e := range events {
		retentionMs := retentionByWorkspace[e.WorkspaceID]
		expiresAt := time.UnixMilli(e.Time + retentionMs)

		base := schema.AuditLogV1{
			EventID: e.ID,
			Time:    e.Time,
			// Every platform event is owned by the Unkey platform workspace,
			// with the originating customer workspace encoded into bucket_id.
			// This keeps the "WHERE workspace_id = me" access model working
			// uniformly once customer-emitted audit logs ship alongside
			// platform events in the same CH table.
			WorkspaceID: unkeyPlatformWorkspaceID,
			BucketID:    unkeyPlatformBucketID(e.WorkspaceID),
			Event:       e.Event,
			Description: e.Display,
			ActorType:   e.ActorType,
			ActorID:     e.ActorID,
			ActorName:   nullString(e.ActorName),
			ActorMeta:   string(e.ActorMeta),
			RemoteIP:    nullString(e.RemoteIp),
			UserAgent:   nullString(e.UserAgent),
			Meta:        "",
			TargetType:  "",
			TargetID:    "",
			TargetName:  "",
			TargetMeta:  "",
			ExpiresAt:   expiresAt,
		}

		targets := targetsByLog[e.ID]
		if len(targets) == 0 {
			out = append(out, base)
			continue
		}
		for _, t := range targets {
			row := base
			row.TargetType = t.Type
			row.TargetID = t.ID
			row.TargetName = nullString(t.Name)
			row.TargetMeta = string(t.Meta)
			out = append(out, row)
		}
	}
	return out
}

// loadRetentionMillis returns a map from workspace ID to the retention
// window (in milliseconds) that should apply to the batch's events. Results
// are cached via s.retentionCh (10m fresh, 1h stale), so a steady-state
// export drains with zero extra MySQL reads once quotas are warm.
func (s *Service) loadRetentionMillis(
	ctx context.Context,
	events []db.FindUnexportedAuditLogsRow,
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

// nullString collapses sqlc's NullString into the plain string the CH schema
// expects. Empty string is our convention for "unset" on the CH side.
func nullString(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return n.String
}
