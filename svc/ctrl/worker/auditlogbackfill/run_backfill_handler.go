package auditlogbackfill

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// batchLimit caps the number of legacy audit_log rows read per page. Each
// row maps to one CH row (targets are folded as Nested arrays inside the
// same row), so this also bounds the CH insert size and the targets IN-list
// length. 5000 sits in CH's INSERT sweet spot and keeps the targets IN-list
// well under Vitess's query-size limits.
const batchLimit int32 = 5000

// maxPagesPerInvocation caps the number of paged iterations a single
// RunBackfill call can chip through. With batchLimit=5000 this means each
// invocation does at most 500k rows (~1-2 min wall-clock). Bounding per-
// invocation work keeps the Restate journal small (one entry per page) so
// crash recovery stays fast, and lets the cron drive throughput by
// scheduling more frequently rather than letting one invocation sprawl
// across hours.
const maxPagesPerInvocation = 100

// stateKeyLastPK is the Restate VO state key for the cursor. Combined
// with cutoff_pk (below) the page query is `WHERE pk > last_pk AND pk <=
// cutoff_pk ORDER BY pk LIMIT ?`, monotonic on the primary key.
const stateKeyLastPK = "last_pk"

// stateKeyCutoffPK is the Restate VO state key for the upper bound on
// what counts as "legacy" rows. Snapshotted on first invocation as
// MAX(pk) of audit_log; rows written after that get shipped via the
// live drainer instead. Without this bound the cursor would chase a
// moving target forever (writers still dual-write to audit_log during
// the dual-write phase).
const stateKeyCutoffPK = "cutoff_pk"

// pageResult is the journaled outcome of one paged backfill iteration.
type pageResult struct {
	RowsBackfilled int32  `json:"rows_backfilled"`
	NewLastPK      uint64 `json:"new_last_pk"`
	Done           bool   `json:"done"`
}

// RunBackfill loops paged scans of the legacy audit_log table past the
// persisted cursor and up to a snapshotted cutoff, ships each page to
// ClickHouse, and advances the cursor in VO state. Each page is its own
// restate.Run so a crash mid-backfill only replays the last incomplete
// page; CH's non_replicated_deduplication_window collapses the duplicate
// insert block.
//
// Termination:
//
//   - Cutoff (MAX(pk) at first invocation) bounds the scan above. Rows
//     written after the snapshot are shipped by the live drainer instead.
//   - Page returns 0 rows OR cursor reaches cutoff: caught up.
//     Response.finished = true. Future ticks are noops (returns immediately).
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

	// Pointer disambiguates "not set yet" (nil) from "snapshotted as 0
	// because audit_log was empty at first run" (non-nil zero). The
	// latter is a real, reachable state in dev environments and on
	// fresh databases — without the pointer we'd re-snapshot forever.
	cutoffPtr, err := restate.Get[*uint64](ctx, stateKeyCutoffPK)
	if err != nil {
		return nil, fmt.Errorf("get cutoff state: %w", err)
	}
	var cutoff uint64
	if cutoffPtr == nil {
		// First invocation: snapshot MAX(pk) so all subsequent runs target
		// the same finite range. Wrapped in restate.Run so the journaled
		// value is replayed deterministically on crash.
		cutoff, err = restate.Run(ctx, func(rc restate.RunContext) (uint64, error) {
			max, err := db.Query.FindAuditLogMaxPK(rc, s.db.RO())
			if err != nil {
				return 0, fmt.Errorf("snapshot max(pk): %w", err)
			}
			return uint64(max), nil
		}, restate.WithName("snapshot cutoff"))
		if err != nil {
			return nil, err
		}
		restate.Set(ctx, stateKeyCutoffPK, &cutoff)
		logger.Info("audit log backfill cutoff snapshotted", "cutoff_pk", cutoff)
	} else {
		cutoff = *cutoffPtr
	}

	var totalBackfilled int32
	var done bool
	if cursor >= cutoff {
		// Already caught up. No-op tick; still send the heartbeat below
		// so silence means "wedged" not "finished."
		done = true
	}
	for pageNum := 0; !done && pageNum < maxPagesPerInvocation; pageNum++ {
		current := cursor
		currentCutoff := cutoff
		page, err := restate.Run(ctx, func(rc restate.RunContext) (pageResult, error) {
			return s.backfillPage(rc, current, currentCutoff)
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
		"cutoff_pk", cutoff,
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

// backfillPage reads one page of legacy audit_log rows past the cursor
// and up to (inclusive) the cutoff, joins their targets in one batched
// query, transforms to v1 CH envelopes, and inserts. Returns Done=true
// when the page is empty (cursor reached cutoff).
//
// Failure modes:
//   - JSON decode of actor_meta or target meta fails: page fails, cursor
//     stays put, manual intervention required (likely a bad legacy row).
//   - CH insert fails: cursor stays put, next tick replays the same page,
//     CH dedup absorbs duplicates if the prior call landed.
func (s *Service) backfillPage(ctx context.Context, afterPK, cutoffPK uint64) (pageResult, error) {
	parents, err := db.Query.FindAuditLogsForBackfill(ctx, s.db.RO(), db.FindAuditLogsForBackfillParams{
		AfterPk:  afterPK,
		CutoffPk: cutoffPK,
		Limit:    batchLimit,
	})
	if err != nil {
		return pageResult{}, fmt.Errorf("find audit logs: %w", err)
	}
	if len(parents) == 0 {
		return pageResult{Done: true, RowsBackfilled: 0, NewLastPK: afterPK}, nil
	}

	parentIDs := make([]string, len(parents))
	for i, p := range parents {
		parentIDs[i] = p.ID
	}

	targetsByLog, err := s.loadTargets(ctx, parentIDs)
	if err != nil {
		return pageResult{}, fmt.Errorf("load targets: %w", err)
	}

	chRows, err := buildCHRows(parents, targetsByLog)
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
//
// inserted_at is set to wall-clock now: the CH table's TTL is keyed off
// inserted_at, so backfilled rows get a full 90 days from when CH saw
// them rather than from the original event time (which would be born
// expired for any row older than 90d).
func buildCHRows(
	parents []db.FindAuditLogsForBackfillRow,
	targetsByLog map[string][]db.FindAuditLogTargetsForBackfillRow,
) ([]schema.AuditLogV1, error) {
	nowMillis := time.Now().UnixMilli()
	out := make([]schema.AuditLogV1, len(parents))
	for i, p := range parents {
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
			EventID:       p.ID,
			Time:          p.Time,
			InsertedAt:    nowMillis,
			WorkspaceID:   p.WorkspaceID,
			Bucket:        p.Bucket,
			Source:        auditlog.EventSourcePlatform,
			Event:         p.Event,
			Description:   p.Display,
			ActorType:     p.ActorType,
			ActorID:       p.ActorID,
			ActorName:     nullStringOrEmpty(p.ActorName),
			ActorMeta:     actorMeta,
			RemoteIP:      nullStringOrEmpty(p.RemoteIp),
			UserAgent:     nullStringOrEmpty(p.UserAgent),
			Meta:          json.RawMessage("{}"),
			TargetTypes:   targetTypes,
			TargetIDs:     targetIDs,
			TargetNames:   targetNames,
			TargetMetas:   targetMetas,
			CorrelationID: "",
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
