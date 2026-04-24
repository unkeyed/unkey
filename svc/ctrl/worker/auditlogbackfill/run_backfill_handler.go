package auditlogbackfill

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

// batchLimit caps the number of legacy audit_log rows read per page. Each
// row maps to one CH row (targets are folded as Nested arrays inside the
// same row), so this also bounds the CH insert size and the targets IN-list
// length.
const batchLimit int32 = 1000

// stateKeyLastPK is the Restate VO state key for the cursor. Storing only
// `last_pk` is enough because the page query is `WHERE pk > last_pk ORDER
// BY pk LIMIT ?`, which is monotonic on the primary key.
const stateKeyLastPK = "last_pk"

// freeTierRetentionMillis mirrors the constant in auditlogexport so a
// backfilled row gets the same expires_at stamp the live drainer would
// have assigned. See svc/ctrl/worker/auditlogexport for the source-of-
// truth comment.
const freeTierRetentionMillis int64 = 30 * 24 * 60 * 60 * 1000

// pageResult is the journaled outcome of one paged backfill iteration.
type pageResult struct {
	RowsBackfilled int32  `json:"rows_backfilled"`
	NewLastPK      uint64 `json:"new_last_pk"`
	Done           bool   `json:"done"`
}

// RunBackfill loops paged scans of the legacy audit_log table past the
// persisted cursor, ships each page to ClickHouse, and advances the
// cursor in VO state. Each page is its own restate.Run so a crash mid-
// backfill only replays the last incomplete page; CH's
// non_replicated_deduplication_window collapses the duplicate insert
// block.
//
// Termination:
//
//   - Page returns 0 rows: cursor caught up with the legacy tail.
//     Response.finished = true. Future ticks are noops.
//   - Page returns < batchLimit rows: still loops one more time so we
//     don't bail early on a half-empty page mid-table; the next page will
//     be empty and trigger the finished branch.
func (s *Service) RunBackfill(
	ctx restate.ObjectContext,
	_ *hydrav1.RunBackfillRequest,
) (*hydrav1.RunBackfillResponse, error) {
	logger.Info("running audit log backfill")
	start := time.Now()

	cursor, err := restate.Get[uint64](ctx, stateKeyLastPK)
	if err != nil {
		return nil, fmt.Errorf("get cursor state: %w", err)
	}

	var totalBackfilled int32
	var done bool
	for pageNum := 0; ; pageNum++ {
		current := cursor
		page, err := restate.Run(ctx, func(rc restate.RunContext) (pageResult, error) {
			return s.backfillPage(rc, current)
		}, restate.WithName(fmt.Sprintf("page-%d", pageNum)))
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", pageNum, err)
		}

		totalBackfilled += page.RowsBackfilled
		if page.RowsBackfilled > 0 {
			cursor = page.NewLastPK
			restate.Set(ctx, stateKeyLastPK, cursor)
		}

		if page.Done {
			done = true
			break
		}
	}

	logger.Info("audit log backfill pass complete",
		"rows_backfilled", totalBackfilled,
		"last_pk", cursor,
		"finished", done,
		"elapsed", time.Since(start),
	)

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunBackfillResponse{
		RowsBackfilled: totalBackfilled,
		LastPk:         cursor,
		Finished:       done,
	}, nil
}

// backfillPage reads one page of legacy audit_log rows past the cursor,
// joins their targets in one batched query, transforms to v1 CH envelopes,
// and inserts. Returns Done=true when the parent page is empty.
//
// Failure modes:
//   - JSON decode of actor_meta or target meta fails: page fails, cursor
//     stays put, manual intervention required (likely a bad legacy row).
//   - CH insert fails: cursor stays put, next tick replays the same page,
//     CH dedup absorbs duplicates if the prior call landed.
func (s *Service) backfillPage(ctx context.Context, afterPK uint64) (pageResult, error) {
	parents, err := db.Query.FindAuditLogsForBackfill(ctx, s.db.RO(), db.FindAuditLogsForBackfillParams{
		AfterPk: afterPK,
		Limit:   batchLimit,
	})
	if err != nil {
		return pageResult{}, fmt.Errorf("find audit logs: %w", err)
	}
	if len(parents) == 0 {
		return pageResult{Done: true}, nil
	}

	parentIDs := make([]string, len(parents))
	for i, p := range parents {
		parentIDs[i] = p.ID
	}

	targetsByLog, err := s.loadTargets(ctx, parentIDs)
	if err != nil {
		return pageResult{}, fmt.Errorf("load targets: %w", err)
	}

	retentionByWorkspace, err := s.loadRetentionMillis(ctx, parents)
	if err != nil {
		return pageResult{}, fmt.Errorf("load retention: %w", err)
	}

	chRows, err := buildCHRows(parents, targetsByLog, retentionByWorkspace)
	if err != nil {
		return pageResult{}, fmt.Errorf("build clickhouse rows: %w", err)
	}

	if err := s.clickhouse.InsertAuditLogs(ctx, chRows); err != nil {
		return pageResult{}, fmt.Errorf("insert clickhouse: %w", err)
	}

	return pageResult{
		RowsBackfilled: int32(len(parents)),
		NewLastPK:      parents[len(parents)-1].Pk,
		Done:           false,
	}, nil
}

// loadTargets does the one batched IN-list lookup that gives us all targets
// for the parent page. The unique index on (audit_log_id, id) covers it;
// rows come back ordered by audit_log_id then pk so the grouping below sees
// a deterministic per-event sequence.
func (s *Service) loadTargets(
	ctx context.Context,
	parentIDs []string,
) (map[string][]db.FindAuditLogTargetsForBackfillRow, error) {
	rows, err := db.Query.FindAuditLogTargetsForBackfill(ctx, s.db.RO(), parentIDs)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]db.FindAuditLogTargetsForBackfillRow, len(parentIDs))
	for _, r := range rows {
		out[r.AuditLogID] = append(out[r.AuditLogID], r)
	}
	return out, nil
}

// buildCHRows maps a page of legacy parents + their targets into v1 CH
// envelopes. Mirrors auditlogexport.buildCHRows so backfilled rows are
// indistinguishable from drained ones.
func buildCHRows(
	parents []db.FindAuditLogsForBackfillRow,
	targetsByLog map[string][]db.FindAuditLogTargetsForBackfillRow,
	retentionByWorkspace map[string]int64,
) ([]schema.AuditLogV1, error) {
	nowMillis := time.Now().UnixMilli()
	out := make([]schema.AuditLogV1, len(parents))
	for i, p := range parents {
		retentionMs := retentionByWorkspace[p.WorkspaceID]

		actorMeta, err := normalizeMeta(p.ActorMeta)
		if err != nil {
			return nil, fmt.Errorf("encode actor_meta pk=%d: %w", p.Pk, err)
		}

		targets := targetsByLog[p.ID]
		targetTypes := make([]string, len(targets))
		targetIDs := make([]string, len(targets))
		targetNames := make([]string, len(targets))
		targetMetas := make([]json.RawMessage, len(targets))
		for j, t := range targets {
			targetTypes[j] = t.Type
			targetIDs[j] = t.ID
			targetNames[j] = nullStringOrEmpty(t.Name)
			tm, err := normalizeMeta(t.Meta)
			if err != nil {
				return nil, fmt.Errorf("encode target_meta pk=%d target_id=%s: %w", p.Pk, t.ID, err)
			}
			targetMetas[j] = tm
		}

		out[i] = schema.AuditLogV1{
			EventID:     p.ID,
			Time:        p.Time,
			InsertedAt:  nowMillis,
			WorkspaceID: p.WorkspaceID,
			Bucket:      p.Bucket,
			Source:      auditlog.EventSourcePlatform,
			Event:       p.Event,
			Description: p.Display,
			ActorType:   p.ActorType,
			ActorID:     p.ActorID,
			ActorName:   nullStringOrEmpty(p.ActorName),
			ActorMeta:   actorMeta,
			RemoteIP:    nullStringOrEmpty(p.RemoteIp),
			UserAgent:   nullStringOrEmpty(p.UserAgent),
			Meta:        json.RawMessage("{}"),
			TargetTypes: targetTypes,
			TargetIDs:   targetIDs,
			TargetNames: targetNames,
			TargetMetas: targetMetas,
			ExpiresAt:   p.Time + retentionMs,
		}
	}
	return out, nil
}

// normalizeMeta turns a legacy json column (which can be SQL NULL or a JSON
// blob) into a JSON object the CH JSON column will accept. Empty / null
// collapse to {} for consistency with the live drainer.
func normalizeMeta(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("{}"), nil
	}
	// Validate by round-trip: legacy rows occasionally hold scalar values
	// in the meta column; CH JSON would reject them.
	var probe any
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	if _, isObj := probe.(map[string]any); !isObj {
		return json.RawMessage("{}"), nil
	}
	return raw, nil
}

func nullStringOrEmpty(s sql.NullString) string {
	if s.Valid {
		return s.String
	}
	return ""
}

// loadRetentionMillis resolves the audit log retention quota for every
// workspace on the page. Mirrors auditlogexport.loadRetentionMillis.
func (s *Service) loadRetentionMillis(
	ctx context.Context,
	parents []db.FindAuditLogsForBackfillRow,
) (map[string]int64, error) {
	workspaces := make(map[string]struct{}, len(parents))
	for _, p := range parents {
		workspaces[p.WorkspaceID] = struct{}{}
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
