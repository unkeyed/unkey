package circuitbreaker

// Metrics defines behavioral methods for observing circuit breaker operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordRequest(name string, state string)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordRequest(string, string) {}
