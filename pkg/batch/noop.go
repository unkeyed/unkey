package batch

import (
	"context"
	"time"

	batchmetrics "github.com/unkeyed/unkey/pkg/batch/metrics"
	buffermetrics "github.com/unkeyed/unkey/pkg/buffer/metrics"
)

// NewNoop creates a minimal no-op BatchProcessor that discards all buffered items.
// It allocates almost no memory (1-element channel with a noop flush) and is safe
// to call from any goroutine. Use this when ClickHouse (or any other sink) is not
// configured but the caller still needs a non-nil *BatchProcessor[T].
func NewNoop[T any]() *BatchProcessor[T] {
	return New(Config[T]{
		Name:          "noop",
		BatchSize:     1,
		BufferSize:    1,
		FlushInterval: time.Hour,
		Drop:          true,
		Consumers:     1,
		Flush:         func(_ context.Context, _ []T) {},
		Metrics:       batchmetrics.NoopMetrics(),
		BufferMetrics: buffermetrics.NoopMetrics(),
	})
}
