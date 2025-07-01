package tracing

import "context"

// Span represents a tracing span
type Span interface {
	End()
	SetAttribute(key string, value any)
	RecordError(err error)
}

// noopSpan is a span that does nothing
type noopSpan struct{}

func (n noopSpan) End()                               {}
func (n noopSpan) SetAttribute(key string, value any) {}
func (n noopSpan) RecordError(err error)              {}

// Start begins a new span with the given name
func Start(ctx context.Context, name string) (context.Context, Span) {
	// Return the same context and a noop span
	return ctx, noopSpan{}
}

// NewSpanName creates a standardized span name
func NewSpanName(service, operation string) string {
	return service + "." + operation
}
