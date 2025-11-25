package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/unkeyed/unkey/go/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/retry"
)

// InsertSettings configures ClickHouse insert behavior.
type InsertSettings struct {
	// AsyncInsert enables async inserts where ClickHouse buffers data server-side.
	AsyncInsert bool
	// WaitForAsyncInsert waits for async insert to flush before returning.
	// Recommended to be true for reliability and backpressure.
	WaitForAsyncInsert bool
	// Deduplicate enables insert deduplication for ReplicatedMergeTree tables.
	// Must be explicitly enabled for async inserts.
	Deduplicate bool
}

// WithInsertSettings applies all insert settings in a single WithSettings call.
// This ensures settings don't overwrite each other.
func WithInsertSettings(s InsertSettings) ch.QueryOption {
	settings := ch.Settings{}

	if s.AsyncInsert {
		settings["async_insert"] = "1"
		if s.WaitForAsyncInsert {
			settings["wait_for_async_insert"] = "1"
		} else {
			settings["wait_for_async_insert"] = "0"
		}
	}

	if s.Deduplicate {
		settings["async_insert_deduplicate"] = "1"
	}

	return ch.WithSettings(settings)
}

// FlushConfig holds configuration for flush operations including retry and circuit breaker.
type FlushConfig struct {
	// CircuitBreaker protects against cascading failures when ClickHouse is unhealthy.
	// Should be shared across all insert types since ClickHouse health affects all tables.
	// If nil, no circuit breaker protection is applied.
	CircuitBreaker *circuitbreaker.CB[struct{}]

	// Retry configures retry behavior for failed flushes.
	// If nil, no retries are attempted.
	Retry *retry.Retry
}

// flush writes a batch of rows to the specified ClickHouse table.
// It handles the preparation of the batch, appending each row as a struct,
// and finally sending the batch to ClickHouse.
//
// This function is used internally by the batch processors to efficiently
// insert data in batches rather than individual rows.
//
// Parameters:
//   - ctx: Context for the operation, allowing for cancellation and timeouts
//   - conn: The ClickHouse connection to use
//   - table: The name of the destination table
//   - rows: A slice of structs representing the rows to insert
//   - cfg: Optional flush configuration for circuit breaker and retry
//   - opts: Optional query options (e.g., ch.WithStdAsync for async inserts)
//
// Returns an error if any part of the batch operation fails.
func flush[T any](ctx context.Context, conn ch.Conn, table string, rows []T, cfg *FlushConfig, opts ...ch.QueryOption) error {
	if len(opts) > 0 {
		ctx = ch.Context(ctx, opts...)
	}

	doFlush := func() error {
		batch, err := conn.PrepareBatch(
			ctx,
			fmt.Sprintf("INSERT INTO %s", table),
			driver.WithReleaseConnection(),
		)
		if err != nil {
			return fault.Wrap(err, fault.Internal("preparing batch failed"))
		}

		for _, row := range rows {
			err = batch.AppendStruct(&row)
			if err != nil {
				return fault.Wrap(err, fault.Internal("appending struct to batch failed"))
			}
		}

		err = batch.Send()
		if err != nil {
			return fault.Wrap(err, fault.Internal("committing batch failed"))
		}

		return nil
	}

	// If no config provided, just execute directly
	if cfg == nil {
		return doFlush()
	}

	// Wrap with retry if configured
	if cfg.Retry != nil {
		originalDoFlush := doFlush
		doFlush = func() error {
			return cfg.Retry.Do(originalDoFlush)
		}
	}

	// Wrap with circuit breaker if configured
	if cfg.CircuitBreaker != nil {
		_, err := cfg.CircuitBreaker.Do(ctx, func(ctx context.Context) (struct{}, error) {
			return struct{}{}, doFlush()
		})

		return err
	}

	return doFlush()
}
