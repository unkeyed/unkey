package keylastusedsync

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// batchSize is the number of keys to fetch from ClickHouse per batch.
// Larger batches reduce CH round trips (the main bottleneck).
const batchSize = 25_000

// updateChunkSize is the number of keys per CASE/WHEN UPDATE statement.
// Benchmarked at ~44K keys/sec with chunk=100 vs ~5K keys/sec for individual UPDATEs.
const updateChunkSize = 100

// PartitionService implements the KeyLastUsedPartitionService Restate virtual object.
// Each instance is keyed by partition index (e.g. "0") and persists its own
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
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil"),
	); err != nil {
		return nil, err
	}

	return &PartitionService{
		UnimplementedKeyLastUsedPartitionServiceServer: hydrav1.UnimplementedKeyLastUsedPartitionServiceServer{},
		db:         cfg.DB,
		clickhouse: cfg.Clickhouse,
	}, nil
}

// batchResult is the journaled output of a single CH→MySQL batch.
// Returned from each restate.Run so Restate can replay without re-executing.
type batchResult struct {
	Synced      int32  `json:"synced"`
	CursorTime  int64  `json:"cursor_time"`
	CursorKeyID string `json:"cursor_key_id"`
}

// SyncPartition reads keys from ClickHouse for this partition and batch-updates MySQL.
// Each batch is a separate restate.Run so that on failure, only the last incomplete
// batch is retried. The cursor is persisted after each batch.
func (s *PartitionService) SyncPartition(
	ctx restate.ObjectContext,
	req *hydrav1.SyncPartitionRequest,
) (*hydrav1.SyncPartitionResponse, error) {
	key := restate.Key(ctx)
	partition, err := strconv.Atoi(key)
	if err != nil {
		return nil, fmt.Errorf("invalid partition key %q: %w", key, err)
	}
	totalPartitions := int(req.GetTotalPartitions())
	if totalPartitions <= 0 {
		return nil, fmt.Errorf("invalid total_partitions: %d", totalPartitions)
	}
	if partition < 0 || partition >= totalPartitions {
		return nil, fmt.Errorf("partition %d out of range [0, %d)", partition, totalPartitions)
	}

	// Read persisted cursor from Restate state.
	// If totalPartitions changed since the last run, the cursor is invalid
	// (hash ranges shifted), so we reset to a full re-sync.
	cursorTime, _ := restate.Get[int64](ctx, "cursor_time")
	cursorKeyID, _ := restate.Get[string](ctx, "cursor_key_id")
	prevTotalPartitions, _ := restate.Get[int](ctx, "total_partitions")

	cursor := clickhouse.KeyLastUsedCursor{Time: cursorTime, KeyID: cursorKeyID}
	if prevTotalPartitions != totalPartitions {
		logger.Info("total_partitions changed, resetting cursor",
			"partition", partition,
			"prev_total_partitions", prevTotalPartitions,
			"new_total_partitions", totalPartitions,
		)
		cursor = clickhouse.KeyLastUsedCursor{Time: 0, KeyID: ""}
		restate.Set(ctx, "total_partitions", totalPartitions)
	}

	logger.Info("partition sync starting",
		"partition", partition,
		"total_partitions", totalPartitions,
		"cursor_time", cursor.Time,
		"cursor_key_id", cursor.KeyID,
	)

	var totalSynced int32
	start := time.Now()

	for batchNum := 0; ; batchNum++ {
		currentCursor := cursor

		result, runErr := restate.Run(ctx, func(rc restate.RunContext) (batchResult, error) {
			chStart := time.Now()
			batch, fetchErr := s.clickhouse.GetKeyLastUsedBatchPartitioned(rc, currentCursor, batchSize, partition, totalPartitions)
			if fetchErr != nil {
				return batchResult{}, fmt.Errorf("fetch partition %d: %w", partition, fetchErr)
			}
			chDur := time.Since(chStart)

			if len(batch) == 0 {
				return batchResult{Synced: 0, CursorTime: 0, CursorKeyID: ""}, nil
			}

			myStart := time.Now()
			if updateErr := s.updateLastUsedBatch(rc, partition, batch); updateErr != nil {
				return batchResult{}, fmt.Errorf("update partition %d: %w", partition, updateErr)
			}
			myDur := time.Since(myStart)

			last := batch[len(batch)-1]

			logger.Info("partition batch complete",
				"partition", partition,
				"batch", batchNum,
				"batch_keys", len(batch),
				"ch_query", chDur,
				"mysql_update", myDur,
				"elapsed", time.Since(start),
			)

			return batchResult{
				Synced:      int32(len(batch)), //nolint:gosec
				CursorTime:  last.Time,
				CursorKeyID: last.KeyID,
			}, nil
		}, restate.WithName(fmt.Sprintf("batch-%d", batchNum)))
		if runErr != nil {
			return nil, fmt.Errorf("partition %d batch %d: %w", partition, batchNum, runErr)
		}

		if result.Synced == 0 {
			break
		}

		totalSynced += result.Synced
		cursor = clickhouse.KeyLastUsedCursor{Time: result.CursorTime, KeyID: result.CursorKeyID}

		// Persist cursor after each batch — on crash we resume from here
		restate.Set(ctx, "cursor_time", cursor.Time)
		restate.Set(ctx, "cursor_key_id", cursor.KeyID)

		if result.Synced < int32(batchSize) { //nolint:gosec
			break
		}
	}

	logger.Info("partition sync complete",
		"partition", partition,
		"keys_synced", totalSynced,
		"cursor_time", cursor.Time,
		"cursor_key_id", cursor.KeyID,
		"elapsed", time.Since(start),
	)

	return &hydrav1.SyncPartitionResponse{
		KeysSynced: totalSynced,
	}, nil
}

func (s *PartitionService) updateLastUsedBatch(ctx context.Context, partition int, batch []clickhouse.KeyLastUsed) error {
	if len(batch) == 0 {
		return nil
	}
	rw := s.db.RW()
	batchStart := time.Now()
	for start := 0; start < len(batch); start += updateChunkSize {
		end := min(start+updateChunkSize, len(batch))
		chunk := batch[start:end]

		if err := s.execCaseUpdate(ctx, rw, chunk); err != nil {
			return fmt.Errorf("update chunk at offset %d: %w", start, err)
		}

		if end%5000 == 0 {
			logger.Info("update progress",
				"partition", partition,
				"keys", end,
				"total", len(batch),
				"elapsed", time.Since(batchStart),
			)
		}
	}
	return nil
}

// buildAndExecCaseUpdate builds and executes a single UPDATE statement like:
//
//	UPDATE `keys` SET last_used_at = CASE id
//	  WHEN 'key1' THEN 1710000000
//	  WHEN 'key2' THEN 1710000001
//	END
//	WHERE id IN ('key1','key2')
//	  AND (last_used_at IS NULL OR last_used_at < CASE id WHEN 'key1' THEN 1710000000 ... END)
func (s *PartitionService) execCaseUpdate(ctx context.Context, rw db.DBTX, chunk []clickhouse.KeyLastUsed) error {
	var b strings.Builder
	// Pre-allocate: each key needs 3 args (CASE, IN, filter CASE)
	args := make([]any, 0, len(chunk)*5+1)

	// SET clause
	b.WriteString("UPDATE `keys` SET last_used_at = CASE id ")
	for _, r := range chunk {
		b.WriteString("WHEN ? THEN ? ")
		args = append(args, r.KeyID, r.Time)
	}
	b.WriteString("END WHERE id IN (")
	for i, r := range chunk {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
		args = append(args, r.KeyID)
	}
	// Only update if new value is actually newer
	b.WriteString(") AND (last_used_at IS NULL OR last_used_at < CASE id ")
	for _, r := range chunk {
		b.WriteString("WHEN ? THEN ? ")
		args = append(args, r.KeyID, r.Time)
	}
	b.WriteString("END)")

	_, err := rw.ExecContext(ctx, b.String(), args...)
	return err
}
