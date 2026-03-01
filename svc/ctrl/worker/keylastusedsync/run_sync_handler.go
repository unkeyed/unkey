package keylastusedsync

import (
	"context"
	"fmt"
	"strings"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

const stateKeyCursorTime = "cursor_time"
const stateKeyCursorKeyID = "cursor_key_id"

// batchSize is the number of keys to fetch from ClickHouse per batch.
const batchSize = 5000

// maxBatchesPerInvocation caps how many batches a single Restate invocation
// processes before returning. This keeps the journal small so replays after
// a crash stay fast. The caller should re-invoke when HasMore is true.
const maxBatchesPerInvocation = 50 // 50 × 500 = 25k keys per invocation

// RunSync reads key last-used timestamps from ClickHouse and batch-updates MySQL.
//
// To keep the Restate journal bounded, each invocation processes at most
// maxBatchesPerInvocation batches (25k keys). The response indicates whether
// more work remains. The caller (e.g. cron) should re-invoke with the same
// key until HasMore is false — the cursor is persisted in Restate state
// across invocations.
func (s *Service) RunSync(
	ctx restate.ObjectContext,
	_ *hydrav1.RunSyncRequest,
) (*hydrav1.RunSyncResponse, error) {
	runKey := restate.Key(ctx)
	logger.Info("running key last used sync", "run", runKey)

	// Restore composite cursor for resumability across invocations
	cursorTime, err := restate.Get[int64](ctx, stateKeyCursorTime)
	if err != nil {
		return nil, fmt.Errorf("get cursor_time state: %w", err)
	}
	cursorKeyID, err := restate.Get[string](ctx, stateKeyCursorKeyID)
	if err != nil {
		return nil, fmt.Errorf("get cursor_key_id state: %w", err)
	}
	cursor := clickhouse.KeyLastUsedCursor{Time: cursorTime, KeyID: cursorKeyID}

	var syncedCount int32
	var hasMore bool

	for batchNum := range maxBatchesPerInvocation {
		batch, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.KeyLastUsed, error) {
			return s.clickhouse.GetKeyLastUsedBatch(rc, cursor, batchSize)
		}, restate.WithName(fmt.Sprintf("fetch ch batch %d", batchNum)))
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch clickhouse batch: %w", fetchErr)
		}

		if len(batch) == 0 {
			break
		}

		_, updateErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, updateLastUsedBatch(rc, s.db, batch)
		}, restate.WithName(fmt.Sprintf("update mysql batch %d", batchNum)))
		if updateErr != nil {
			return nil, fmt.Errorf("update mysql batch: %w", updateErr)
		}

		// Advance cursor to the last element — results are ordered by (time, key_id)
		last := batch[len(batch)-1]
		cursor = clickhouse.KeyLastUsedCursor{Time: last.Time, KeyID: last.KeyID}
		syncedCount += int32(len(batch)) //nolint:gosec

		restate.Set(ctx, stateKeyCursorTime, cursor.Time)
		restate.Set(ctx, stateKeyCursorKeyID, cursor.KeyID)

		if len(batch) == batchSize && batchNum == maxBatchesPerInvocation-1 {
			hasMore = true
		}

		if len(batch) < batchSize {
			break
		}
	}

	logger.Info("key last used sync complete",
		"run", runKey,
		"keys_synced", syncedCount,
		"has_more", hasMore,
		"cursor_time", cursor.Time,
		"cursor_key_id", cursor.KeyID,
	)

	if !hasMore {
		_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
			return restate.Void{}, s.heartbeat.Ping(rc)
		}, restate.WithName("send heartbeat"))
		if err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
	}

	return &hydrav1.RunSyncResponse{
		KeysSynced: syncedCount,
		HasMore:    hasMore,
	}, nil
}

// updateLastUsedBatch builds and executes a single UPDATE with CASE-WHEN to set
// last_used_at for multiple keys at once. It only updates when the new value is
// actually newer than the existing one (idempotent).
func updateLastUsedBatch(ctx context.Context, database db.Database, batch []clickhouse.KeyLastUsed) error {
	if len(batch) == 0 {
		return nil
	}

	// Each key appears 3 times: SET CASE, WHERE IN, guard CASE
	args := make([]any, 0, len(batch)*6)

	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE `keys` SET last_used_at = CASE id ")

	for _, r := range batch {
		queryBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, r.KeyID, r.Time)
	}
	queryBuilder.WriteString("ELSE last_used_at END WHERE id IN (")

	for i, r := range batch {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		queryBuilder.WriteString("?")
		args = append(args, r.KeyID)
	}

	queryBuilder.WriteString(") AND (last_used_at IS NULL OR last_used_at < CASE id ")

	for _, r := range batch {
		queryBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, r.KeyID, r.Time)
	}
	queryBuilder.WriteString("ELSE last_used_at END)")

	_, err := database.RW().ExecContext(ctx, queryBuilder.String(), args...)

	return err
}
