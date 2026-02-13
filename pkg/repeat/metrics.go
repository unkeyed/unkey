package repeat

// Metrics defines behavioral methods for observing repeat operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordPanic(caller, path string)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordPanic(string, string) {}
