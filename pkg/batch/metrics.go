package batch

// Metrics defines behavioral methods for observing batch operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordFlush(name, trigger string, batchSize int)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordFlush(string, string, int) {}
