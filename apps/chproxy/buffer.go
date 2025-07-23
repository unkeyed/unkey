package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// startBufferProcessor manages processing of data sent to Clickhouse.
// Returns a channel that signals when all pending batches have been processed during shutdown
func startBufferProcessor(
	ctx context.Context,
	buffer <-chan *Batch,
	config *Config,
	telemetryConfig *TelemetryConfig,
) <-chan bool {
	done := make(chan bool)

	go func() {
		buffered := 0
		batchesByParams := make(map[string]*Batch)
		ticker := time.NewTicker(config.FlushInterval)
		defer ticker.Stop()

		flushAndReset := func(ctx context.Context, reason string) {
			reason = fmt.Sprintf("flush_batches.%s", reason)
			ctx, span := telemetryConfig.Tracer.Start(ctx, reason)
			defer span.End()

			startTime := time.Now()

			telemetryConfig.Metrics.FlushCounter.Add(ctx, 1)
			telemetryConfig.Metrics.FlushBatchCount.Record(ctx, int64(len(batchesByParams)))

			span.SetAttributes(
				attribute.Int("batch_count", len(batchesByParams)),
				attribute.Int("buffered_rows", buffered),
			)

			// Process batches in parallel for better throughput
			var wg sync.WaitGroup
			for _, batch := range batchesByParams {
				wg.Add(1)
				go func(b *Batch) {
					defer wg.Done()

					batchStart := time.Now()
					err := persist(ctx, b, config)
					batchDuration := time.Since(batchStart).Seconds()
					telemetryConfig.Metrics.BatchPersistDuration.Record(ctx, batchDuration)

					if err != nil {
						span.RecordError(err)
						span.SetStatus(codes.Error, err.Error())

						config.Logger.Error("error persisting batch, data dropped",
							"error", err.Error(),
							"table", b.Table,
							"rows_dropped", len(b.Rows),
							"batch_duration_seconds", batchDuration,
							"query", b.Params.Get("query"),
						)
					}
				}(batch)
			}
			wg.Wait() // Wait for all parallel persists to complete

			duration := time.Since(startTime).Seconds()
			telemetryConfig.Metrics.FlushDuration.Record(ctx, duration)

			span.SetAttributes(attribute.Float64("duration_seconds", duration))
			span.SetStatus(codes.Ok, "")

			buffered = 0
			SetBufferSize(0)
			batchesByParams = make(map[string]*Batch)
		}

		for {
			select {
			case <-ctx.Done():
				config.Logger.Info("context cancelled, flushing remaining batches")
				flushAndReset(ctx, "context_cancelled")
				done <- true
				return
			case b, ok := <-buffer:
				if !ok {
					config.Logger.Info("buffer channel closed, flushing remaining batches")
					flushAndReset(ctx, "buffer_closed")
					done <- true
					return
				}

				params := b.Params.Encode()
				batch, ok := batchesByParams[params]
				if !ok {
					batchesByParams[params] = b
					config.Logger.Info("new batch type received",
						"query", b.Params.Get("query"),
						"table", b.Table,
						"params", params,
						"total_batch_types", len(batchesByParams))
				} else {
					// Check if adding these rows would exceed the per-batch limit
					if len(batch.Rows)+len(b.Rows) > maxBatchRows {
						// Flush the current batch to make room
						config.Logger.Info("flushing batch due to individual batch size limit",
							"current_rows", len(batch.Rows),
							"incoming_rows", len(b.Rows),
							"table", batch.Table)

						err := persist(ctx, batch, config)

						// Always free the memory by resetting batch, regardless of persist success
						// Update buffered count: subtract old rows, will add new rows below
						buffered -= len(batch.Rows)

						if err != nil {
							config.Logger.Error("error persisting batch during size limit flush, data dropped",
								"error", err.Error(),
								"table", batch.Table,
								"rows_dropped", len(batch.Rows),
								"query", batch.Params.Get("query"))
						}

						// Reset this batch and start fresh with the new rows
						batch.Rows = b.Rows
					} else {
						batch.Rows = append(batch.Rows, b.Rows...)
					}
				}

				// Always add the new incoming rows to the buffer count
				buffered += len(b.Rows)
				SetBufferSize(int64(buffered))

				if buffered >= maxBatchSize {
					config.Logger.Info("flushing due to max batch size")
					flushAndReset(ctx, "max_batch_size")
				}
			case <-ticker.C:
				config.Logger.Info("flushing on ticker",
					"buffered_rows", buffered)
				flushAndReset(ctx, "ticker")
			}
		}
	}()

	return done
}
