package keylastusedsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// batchSize is the number of keys to fetch from ClickHouse per batch.
// Larger batches reduce CH round trips (the main bottleneck).
const batchSize = 25_000

// PartitionService implements the KeyLastUsedPartitionService Restate virtual object.
// Each instance is keyed by partition ID (e.g. "partition-0") and persists its own
// cursor in Restate state so that subsequent invocations only process new data.
type PartitionService struct {
	hydrav1.UnimplementedKeyLastUsedPartitionServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
}

var _ hydrav1.KeyLastUsedPartitionServiceServer = (*PartitionService)(nil)

// PartitionConfig holds the configuration for the partition service.
type PartitionConfig struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
}

// NewPartitionService creates a new partition service.
func NewPartitionService(cfg PartitionConfig) (*PartitionService, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("DB must not be nil")
	}
	if cfg.Clickhouse == nil {
		return nil, fmt.Errorf("Clickhouse must not be nil")
	}
	return &PartitionService{
		UnimplementedKeyLastUsedPartitionServiceServer: hydrav1.UnimplementedKeyLastUsedPartitionServiceServer{},
		db:         cfg.DB,
		clickhouse: cfg.Clickhouse,
	}, nil
}

// SyncPartition reads keys from ClickHouse for this partition and batch-updates MySQL.
// The cursor is persisted in Restate state between invocations so that only new data
// is processed on subsequent runs.
func (s *PartitionService) SyncPartition(
	ctx restate.ObjectContext,
	req *hydrav1.SyncPartitionRequest,
) (*hydrav1.SyncPartitionResponse, error) {
	partition := int(req.GetPartition())
	key := restate.Key(ctx)

	// Read persisted cursor from Restate state
	cursorTime, _ := restate.Get[int64](ctx, "cursor_time")
	cursorKeyID, _ := restate.Get[string](ctx, "cursor_key_id")
	cursor := clickhouse.KeyLastUsedCursor{Time: cursorTime, KeyID: cursorKeyID}

	logger.Info("partition sync starting",
		"key", key,
		"partition", partition,
		"total_partitions", defaultPartitions,
		"cursor_time", cursor.Time,
		"cursor_key_id", cursor.KeyID,
	)

	var synced int32

	// Run the actual sync in a single RunAsync so the CH+MySQL work is a side effect
	result, err := restate.Run(ctx, func(rc restate.RunContext) (syncResult, error) {
		return runPartitionSync(rc, s.clickhouse, s.db, cursor, partition, defaultPartitions)
	}, restate.WithName("sync"))
	if err != nil {
		return nil, fmt.Errorf("partition %d sync: %w", partition, err)
	}

	synced = result.Synced

	// Persist the new cursor for the next invocation
	if result.Synced > 0 {
		restate.Set(ctx, "cursor_time", result.CursorTime)
		restate.Set(ctx, "cursor_key_id", result.CursorKeyID)
	}

	logger.Info("partition sync complete",
		"key", key,
		"partition", partition,
		"keys_synced", synced,
		"new_cursor_time", result.CursorTime,
		"new_cursor_key_id", result.CursorKeyID,
	)

	return &hydrav1.SyncPartitionResponse{
		KeysSynced: synced,
	}, nil
}

// syncResult carries the output of a partition sync back through Restate's Run.
type syncResult struct {
	Synced      int32  `json:"synced"`
	CursorTime  int64  `json:"cursor_time"`
	CursorKeyID string `json:"cursor_key_id"`
}

// runPartitionSync pages through ClickHouse for the given partition and batch-updates MySQL.
func runPartitionSync(
	ctx context.Context,
	ch clickhouse.ClickHouse,
	database db.Database,
	cursor clickhouse.KeyLastUsedCursor,
	partition, totalPartitions int,
) (syncResult, error) {
	var synced int32
	currentCursor := cursor
	batchNum := 0
	start := time.Now()

	for {
		chStart := time.Now()
		batch, err := ch.GetKeyLastUsedBatchPartitioned(ctx, currentCursor, batchSize, partition, totalPartitions)
		if err != nil {
			return syncResult{}, fmt.Errorf("fetch partition %d: %w", partition, err)
		}
		chDur := time.Since(chStart)

		if len(batch) == 0 {
			break
		}

		myStart := time.Now()
		if err := updateLastUsedBatch(ctx, database, batch); err != nil {
			return syncResult{}, fmt.Errorf("update partition %d: %w", partition, err)
		}
		myDur := time.Since(myStart)

		synced += int32(len(batch)) //nolint:gosec
		batchNum++

		// Advance cursor to the last element — results are ordered by (time, key_id)
		last := batch[len(batch)-1]
		currentCursor = clickhouse.KeyLastUsedCursor{Time: last.Time, KeyID: last.KeyID}

		logger.Info("partition batch complete",
			"partition", partition,
			"batch", batchNum,
			"batch_keys", len(batch),
			"total_synced", synced,
			"ch_query", chDur,
			"mysql_update", myDur,
			"elapsed", time.Since(start),
		)

		if len(batch) < batchSize {
			break
		}
	}

	return syncResult{
		Synced:      synced,
		CursorTime:  currentCursor.Time,
		CursorKeyID: currentCursor.KeyID,
	}, nil
}

// maxKeysPerUpdate is the max keys in a single UPDATE statement.
// Kept small (2K) so row locks are held briefly and 8 concurrent partition
// writers don't cause lock wait timeouts. 2K × 6 = 12K placeholders.
const maxKeysPerUpdate = 2_000

// updateLastUsedBatch builds and executes CASE-WHEN UPDATEs to set last_used_at
// for multiple keys at once, chunking to stay within MySQL's placeholder limit.
// Only updates when the new value is actually newer (idempotent).
func updateLastUsedBatch(ctx context.Context, database db.Database, batch []clickhouse.KeyLastUsed) error {
	for i := 0; i < len(batch); i += maxKeysPerUpdate {
		end := min(i+maxKeysPerUpdate, len(batch))
		if err := updateLastUsedChunk(ctx, database, batch[i:end]); err != nil {
			return err
		}
	}
	return nil
}

func updateLastUsedChunk(ctx context.Context, database db.Database, chunk []clickhouse.KeyLastUsed) error {
	if len(chunk) == 0 {
		return nil
	}

	args := make([]any, 0, len(chunk)*6)

	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE `keys` SET last_used_at = CASE id ")

	for _, r := range chunk {
		queryBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, r.KeyID, r.Time)
	}
	queryBuilder.WriteString("ELSE last_used_at END WHERE id IN (")

	for i, r := range chunk {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		queryBuilder.WriteString("?")
		args = append(args, r.KeyID)
	}

	queryBuilder.WriteString(") AND (last_used_at IS NULL OR last_used_at < CASE id ")

	for _, r := range chunk {
		queryBuilder.WriteString("WHEN ? THEN ? ")
		args = append(args, r.KeyID, r.Time)
	}
	queryBuilder.WriteString("ELSE last_used_at END)")

	_, err := database.RW().ExecContext(ctx, queryBuilder.String(), args...)
	return err
}
