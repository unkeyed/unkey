package clickhouse

import (
	"context"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// InsertQuery builds "INSERT INTO <table> (<columns>)" from T's generated
// table name and column list (see schema.Row). Naming the columns explicitly
// means the server fills omitted columns from their DEFAULT expressions, so a
// binary whose struct predates a newly added table column keeps flushing
// instead of failing AppendStruct with "missing destination name". Use this
// for every PrepareBatch that appends T via AppendStruct.
func InsertQuery[T schema.Row]() string {
	var row T
	return "INSERT INTO " + row.Table() + " (" + row.InsertColumns() + ")"
}

// flush writes a batch of rows to T's ClickHouse table.
// It automatically applies:
//   - Async insert settings (async_insert=1, wait_for_async_insert=1, async_insert_deduplicate=1)
//   - Retry with exponential backoff (5 attempts)
//   - Circuit breaker protection
//
// Returns an error if any part of the batch operation fails after all retries.
func flush[T schema.Row](c *Client, ctx context.Context, rows []T) error {
	// Apply async insert settings
	ctx = ch.Context(ctx, ch.WithSettings(ch.Settings{
		"async_insert":             "1",
		"wait_for_async_insert":    "1",
		"async_insert_deduplicate": "1",
	}))

	query := InsertQuery[T]()

	doFlush := func() error {
		batch, err := c.conn.PrepareBatch(
			ctx,
			query,
			driver.WithReleaseConnection(),
		)
		if err != nil {
			return fault.Wrap(err, fault.Internal("preparing batch failed"))
		}
		defer func() {
			if err = batch.Close(); err != nil {
				logger.Error("failed to close batch", "error", err.Error())
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
