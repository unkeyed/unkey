package main

import (
	"context"
	"fmt"
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

			span.SetAttributes(
				attribute.Int("batch_count", len(batchesByParams)),
				attribute.Int("buffered_rows", buffered),
			)

			for _, batch := range batchesByParams {
				err := persist(ctx, batch, config)
				if err != nil {
					span.RecordError(err)
					span.SetStatus(codes.Error, err.Error())

					config.Logger.Error("error persisting batch",
						"error", err.Error(),
						"table", batch.Table,
						"query", batch.Params.Get("query"),
					)
				}
			}

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
					config.Logger.Debug("new batch type received",
						"query", b.Params.Get("query"))
				} else {
					batch.Rows = append(batch.Rows, b.Rows...)
				}

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
