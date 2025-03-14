package main

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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

		tickerCount := 0

		flushAndReset := func(ctx context.Context, reason string) {
			ctx, span := telemetryConfig.Tracer.Start(ctx, "flush_batches")
			defer span.End()

			startTime := time.Now()

			// Record metrics
			telemetryConfig.Metrics.FlushCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))

			span.SetAttributes(
				attribute.Int("batch_count", len(batchesByParams)),
				attribute.Int("buffered_rows", buffered),
				attribute.String("reason", reason),
			)

			// We'll sample logs for ticker-based flushes
			shouldLog := true
			if reason == "ticker" {
				tickerCount++
				// Only log every LOG_TICKER_SAMPLE_RATE times
				shouldLog = (tickerCount%LOG_TICKER_SAMPLE_RATE == 0)
			}

			// Only log if we should based on sampling
			if shouldLog {
				config.Logger.Info("flushing batches",
					"reason", reason,
					"batch_count", len(batchesByParams),
					"buffered_rows", buffered)
			}

			for _, batch := range batchesByParams {
				err := persist(ctx, batch, config)
				if err != nil {
					// Always log errors regardless of sampling
					config.Logger.Error("error flushing batch",
						"error", err.Error(),
						"query", batch.Params.Get("query"))
				}
			}

			duration := time.Since(startTime).Seconds()
			telemetryConfig.Metrics.FlushDuration.Record(ctx, duration,
				metric.WithAttributes(attribute.String("reason", reason)))

			span.SetAttributes(attribute.Float64("duration_seconds", duration))

			buffered = 0
			SetBufferSize(0)
			batchesByParams = make(map[string]*Batch)
		}

		for {
			select {
			case <-ctx.Done():
				config.Logger.Info("context cancelled, flushing remaining batches",
					"buffered_rows", buffered,
					"elapsed_time", config.FlushInterval.String())
				flushAndReset(ctx, "shutdown")
				done <- true
				return
			case b, ok := <-buffer:
				if !ok {
					config.Logger.Info("buffer channel closed, flushing remaining batches")
					flushAndReset(ctx, "shutdown")
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

				if buffered >= config.MaxBatchSize {
					config.Logger.Info("flushing due to max batch size",
						"buffered_rows", buffered,
						"max_size", config.MaxBatchSize)
					flushAndReset(ctx, "max_size")
				}
			case <-ticker.C:
				config.Logger.Info("flushing on ticker",
					"buffered_rows", buffered,
					"elapsed_time", config.FlushInterval.String())
				flushAndReset(ctx, "ticker")
			}
		}
	}()

	return done
}
