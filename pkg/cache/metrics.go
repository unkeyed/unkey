package cache

// Metrics defines behavioral methods for observing cache operations.
// Implementations must be safe for concurrent use.
type Metrics interface {
	RecordRead(resource string, hit bool)
	RecordDeleted(resource, reason string)
	RecordRevalidation(resource string, count float64)
	RecordSizeAndCapacity(resource string, size, capacity float64)
}

// NoopMetrics is a no-op implementation of Metrics for use in tests.
type NoopMetrics struct{}

func (NoopMetrics) RecordRead(string, bool)                   {}
func (NoopMetrics) RecordDeleted(string, string)              {}
func (NoopMetrics) RecordRevalidation(string, float64)        {}
func (NoopMetrics) RecordSizeAndCapacity(string, float64, float64) {}
