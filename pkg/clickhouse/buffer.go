package clickhouse

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/logger"
)

// BufferConfig configures a batch buffer created via NewBuffer.
type BufferConfig struct {
	// Name identifies this buffer for logging and metrics.
	Name string

	// BatchSize is the maximum number of items to collect before flushing.
	BatchSize int

	// BufferSize is the capacity of the channel buffer holding incoming items.
	BufferSize int

	// FlushInterval is the maximum time to wait before flushing a non-empty batch.
	FlushInterval time.Duration

	// Consumers specifies how many goroutine workers should process the channel.
	Consumers int

	// Drop determines whether to discard items when the buffer is full.
	// When true, new items are silently dropped if the buffer is at capacity.
	// When false (default), Buffer() will block until space becomes available.
	Drop bool

	// OnFlushError is called when a flush fails. If nil, errors are logged
	// via logger.Error (best-effort). Callers that need strict error handling
	// can supply their own callback.
	OnFlushError func(ctx context.Context, table string, rowCount int, err error)
}

// NewBuffer creates a *batch.BatchProcessor[T] that flushes rows to the given
// ClickHouse table using the client's connection, retry policy, and circuit breaker.
//
// The caller owns the returned processor and must call Close() on it during shutdown
// (before closing the Client) to drain any buffered rows.
//
// Example:
//
//	buf := clickhouse.NewBuffer[schema.SentinelRequest](client, "default.sentinel_requests_raw_v1", clickhouse.BufferConfig{
//	    Name:          "sentinel_requests",
//	    BatchSize:     50_000,
//	    BufferSize:    50_000,
//	    FlushInterval: 5 * time.Second,
//	    Consumers:     2,
//	})
//	defer buf.Close()
func NewBuffer[T any](c *Client, table string, cfg BufferConfig) *batch.BatchProcessor[T] {
	onErr := cfg.OnFlushError
	if onErr == nil {
		onErr = func(_ context.Context, tbl string, _ int, err error) {
			logger.Error("failed to flush batch", "table", tbl, "error", err.Error())
		}
	}

	return batch.New(batch.Config[T]{
		Name:          cfg.Name,
		Drop:          cfg.Drop,
		BatchSize:     cfg.BatchSize,
		BufferSize:    cfg.BufferSize,
		FlushInterval: cfg.FlushInterval,
		Consumers:     cfg.Consumers,
		Flush: func(ctx context.Context, rows []T) {
			if err := flush(c, ctx, table, rows); err != nil {
				onErr(ctx, table, len(rows), err)
			}
		},
	})
}
