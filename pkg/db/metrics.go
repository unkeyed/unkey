package db

// Metrics defines behavioral methods for observing database operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordOperation(replica, operation, status string, duration float64)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordOperation(string, string, string, float64) {}
