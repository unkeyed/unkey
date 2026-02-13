package zen

// Metrics defines behavioral methods for observing HTTP request handling.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordHTTPRequest(method, path string, status int, bodySize int, latency float64)
	RecordPanic(caller, path string)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordHTTPRequest(string, string, int, int, float64) {}
func (NoopMetrics) RecordPanic(string, string)                          {}
