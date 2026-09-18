package clickhouse

import (
	"context"
	"errors"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const blockBodyBytesMax = 1 << 20

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
	ctx, span := tracing.Start(ctx, "clickhouse.flush")
	defer span.End()
	var row T
	span.SetAttributes(attribute.String("db.collection.name", row.Table()), attribute.Int("rows", len(rows)))
	// Apply async insert settings
	ctx = ch.Context(ctx, ch.WithSettings(ch.Settings{
		"async_insert":             "1",
		"wait_for_async_insert":    "1",
		"async_insert_deduplicate": "1",
	}))

	query := InsertQuery[T]()
	attempt := 0

	doFlush := func() (attemptErr error) {
		attempt++
		ctx, attemptSpan := tracing.Start(ctx, "clickhouse.insert")
		defer attemptSpan.End()
		stats := c.conn.Stats()
		attemptSpan.SetAttributes(attribute.Int("attempt", attempt),
			attribute.Int("pool.open", stats.Open), attribute.Int("pool.idle", stats.Idle),
			attribute.Int("pool.max", stats.MaxOpenConns))
		defer func() {
			tracing.RecordError(attemptSpan, attemptErr)
		}()
		attemptSpan.AddEvent("prepare")
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

		attemptSpan.AddEvent("append")
		if err = appendRows(batch, rows); err != nil {
			if abortErr := batch.Abort(); abortErr != nil {
				return errors.Join(err, fault.Wrap(abortErr, fault.Internal("aborting batch failed")))
			}
			return err
		}

		attemptSpan.AddEvent("send")
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
	tracing.RecordError(span, err)

	return err
}

func appendRows[T schema.Row](batch driver.Batch, rows []T) error {
	blockBodyBytes := 0
	for i := range rows {
		if request, ok := any(rows[i]).(schema.FrontlineRequest); ok {
			bodyBytes := len(request.RequestBody) + len(request.ResponseBody)
			if blockBodyBytes > 0 && bodyBytes > blockBodyBytesMax-blockBodyBytes {
				if err := batch.Flush(); err != nil {
					return fault.Wrap(err, fault.Internal("flushing body block failed"))
				}
				blockBodyBytes = 0
			}
			blockBodyBytes += bodyBytes
		}
		if err := batch.AppendStruct(&rows[i]); err != nil {
			return fault.Wrap(err, fault.Internal("appending struct to batch failed"))
		}
	}
	return nil
}
