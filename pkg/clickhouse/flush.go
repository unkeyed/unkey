package clickhouse

import (
	"context"
	"time"

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
	var row T
	started := time.Now()
	attempt := 0

	doFlush := func() (attemptErr error) {
		attempt++
		start := time.Now()
		stage := "prepare"
		var prepare, appendTime, send time.Duration
		before := c.conn.Stats()
		if c.logInsertTimings {
			logger.Info("clickhouse insert attempt started", "table", row.Table(), "attempt", attempt,
				"rows", len(rows), "pool_open", before.Open, "pool_idle", before.Idle, "pool_max", before.MaxOpenConns)
		}
		defer func() {
			if !c.logInsertTimings {
				return
			}
			after := c.conn.Stats()
			fields := []any{"table", row.Table(), "attempt", attempt, "rows", len(rows), "stage", stage,
				"elapsed_ms", time.Since(start).Milliseconds(), "prepare_ms", prepare.Milliseconds(),
				"append_ms", appendTime.Milliseconds(), "send_ms", send.Milliseconds(),
				"pool_open_before", before.Open, "pool_idle_before", before.Idle,
				"pool_open", after.Open, "pool_idle", after.Idle, "pool_max", after.MaxOpenConns}
			if attemptErr != nil {
				logger.Warn("clickhouse insert attempt failed", append(fields, "error", attemptErr.Error())...)
			} else {
				logger.Info("clickhouse insert attempt complete", fields...)
			}
		}()
		batch, err := c.conn.PrepareBatch(
			ctx,
			query,
			driver.WithReleaseConnection(),
		)
		prepare = time.Since(start)
		if err != nil {
			return fault.Wrap(err, fault.Internal("preparing batch failed"))
		}
		defer func() {
			if err = batch.Close(); err != nil {
				logger.Error("failed to close batch", "error", err.Error())
			}
		}()

		stage = "append"
		stageStart := time.Now()
		for _, row := range rows {
			err = batch.AppendStruct(&row)
			if err != nil {
				appendTime = time.Since(stageStart)
				return fault.Wrap(err, fault.Internal("appending struct to batch failed"))
			}
		}
		appendTime = time.Since(stageStart)

		stage = "send"
		stageStart = time.Now()
		err = batch.Send()
		send = time.Since(stageStart)
		if err != nil {
			return fault.Wrap(err, fault.Internal("committing batch failed"))
		}

		return nil
	}

	// Wrap with retry, then circuit breaker
	_, err := c.circuitBreaker.Do(ctx, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, c.retry.DoContext(ctx, doFlush)
	})
	if c.logInsertTimings {
		logger.Info("clickhouse flush finished", "table", row.Table(), "rows", len(rows),
			"attempts", attempt, "elapsed_ms", time.Since(started).Milliseconds(), "error", err)
	}

	return err
}
