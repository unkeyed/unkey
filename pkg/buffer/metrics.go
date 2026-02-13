package buffer

// Metrics defines behavioral methods for observing buffer operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordState(name, state string)
	RecordSize(name string, drop bool, ratio float64)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordState(string, string)          {}
func (NoopMetrics) RecordSize(string, bool, float64) {}
