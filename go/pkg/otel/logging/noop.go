package logging

import (
	"context"
	"log/slog"
)

// noop implements a Logger that discards all log messages.
// It's useful for testing environments or when logging needs to be completely disabled.
type noop struct {
}

// NewNoop creates a new no-op logger that discards all log messages.
// This is particularly useful in testing scenarios where logs would
// add noise to test output, or in benchmarks where logging overhead
// should be eliminated.
//
// Example:
//
//	// In tests
//	func TestSomething(t *testing.T) {
//	    // Use a no-op logger to keep test output clean
//	    logger := logging.NewNoop()
//	    service := myservice.New(myservice.Config{Logger: logger})
//	    // ...
//	}
func NewNoop() Logger {
	return &noop{}
}

// With returns the same noop logger, ignoring the arguments.
func (l *noop) With(args ...any) Logger {
	return l
}

// WithAttrs returns the same noop logger, ignoring the attributes.
func (l *noop) WithAttrs(attrs ...slog.Attr) Logger {
	return l
}

// ---- Standard logging no-op methods ----

// Debug is a no-op implementation that discards debug level messages.
func (l *noop) Debug(msg string, args ...any) {
}

// Info is a no-op implementation that discards info level messages.
func (l *noop) Info(msg string, args ...any) {
}

// Warn is a no-op implementation that discards warn level messages.
func (l *noop) Warn(msg string, args ...any) {
}

// Error is a no-op implementation that discards error level messages.
func (l *noop) Error(msg string, args ...any) {
}

// ---- Context-aware logging no-op methods ----

// DebugContext is a no-op implementation that discards debug level messages.
func (l *noop) DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
}

// InfoContext is a no-op implementation that discards info level messages.
func (l *noop) InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
}

// WarnContext is a no-op implementation that discards warn level messages.
func (l *noop) WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
}

// ErrorContext is a no-op implementation that discards error level messages.
func (l *noop) ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
}
