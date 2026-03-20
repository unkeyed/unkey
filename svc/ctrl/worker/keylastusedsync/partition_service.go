package keylastusedsync

import (
	"context"
	"fmt"
	"strconv"
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

// minuteMillis is used to truncate timestamps to minute granularity.
// Keys used in the same minute get the same last_used_at, allowing simple
// bulk UPDATEs grouped by timestamp instead of per-key CASE/WHEN.
const minuteMillis = 60_000

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
	if prevTotalPartitions > 0 && prevTotalPartitions != totalPartitions {
		logger.Info("total_partitions changed, resetting cursor",
			"partition", partition,
			"prev_total_partitions", prevTotalPartitions,
			"new_total_partitions", totalPartitions,
		)
		cursor = clickhouse.KeyLastUsedCursor{Time: 0, KeyID: ""}
	}
	restate.Set(ctx, "total_partitions", totalPartitions)

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
			batch, fetchErr := s.clickhouse.GetKeyLastUsedBatchPartitioned(rc, clickhouse.GetKeyLastUsedBatchRequest{
				Cursor:          currentCursor,
				Limit:           batchSize,
				Partition:       partition,
				TotalPartitions: totalPartitions,
			})
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

	// Group keys by minute-truncated timestamp. Keys used in the same minute
	// share a single UPDATE ... WHERE id IN (...) statement.
	groups := make(map[int64][]string)
	for _, r := range batch {
		minute := (r.Time / minuteMillis) * minuteMillis
		groups[minute] = append(groups[minute], r.KeyID)
	}

	rw := s.db.RW()
	batchStart := time.Now()
	done := 0
	for ts, keyIDs := range groups {
		if _, err := db.WithRetryContext(ctx, func() (struct{}, error) {
			return struct{}{}, db.Query.UpdateKeysLastUsed(ctx, rw, db.UpdateKeysLastUsedParams{
				LastUsedAt: uint64(ts), //nolint:gosec
				KeyIds:     keyIDs,
			})
		}); err != nil {
			return fmt.Errorf("update minute %d: %w", ts, err)
		}
		done += len(keyIDs)
		if done%5000 < len(keyIDs) {
			logger.Info("update progress",
				"partition", partition,
				"keys", done,
				"total", len(batch),
				"groups", len(groups),
				"elapsed", time.Since(batchStart),
			)
		}
	}
	return nil
}
