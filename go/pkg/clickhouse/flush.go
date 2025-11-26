package clickhouse

import (
	"context"
	"fmt"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// flush writes a batch of rows to the specified ClickHouse table.
// It automatically applies:
//   - Async insert settings (async_insert=1, wait_for_async_insert=1, async_insert_deduplicate=1)
//   - Retry with exponential backoff (5 attempts)
//   - Circuit breaker protection
//
// Returns an error if any part of the batch operation fails after all retries.
func flush[T any](c *clickhouse, ctx context.Context, table string, rows []T) error {
	// Apply async insert settings
	ctx = ch.Context(ctx, ch.WithSettings(ch.Settings{
		"async_insert":             "1",
		"wait_for_async_insert":    "1",
		"async_insert_deduplicate": "1",
	}))

	doFlush := func() error {
		batch, err := c.conn.PrepareBatch(
			ctx,
			fmt.Sprintf("INSERT INTO %s", table),
			driver.WithReleaseConnection(),
		)
		if err != nil {
			return fault.Wrap(err, fault.Internal("preparing batch failed"))
		}
		defer func() {
			if err := batch.Close(); err != nil {
				c.logger.Error("failed to close batch", "error", err.Error())
			}
		}()

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

	// Wrap with retry, then circuit breaker
	_, err := c.circuitBreaker.Do(ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.retry.DoContext(ctx, doFlush)
	})

	return err
}
