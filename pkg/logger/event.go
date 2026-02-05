package logger

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// eventKey is the context key for storing wide events.
type eventKey struct{}

// Event accumulates attributes and errors throughout a request lifecycle,
// emitting a single log entry when [End] is called. This "wide event" pattern
// reduces log volume while preserving full context for debugging.
//
// Safe for concurrent use. Multiple goroutines can call [Event.Add] and
// [Event.SetError] on the same event.
type Event struct {
	mu       sync.Mutex
	start    time.Time
	duration time.Duration
	attrs    []slog.Attr
	errors   []error
	written  bool
}

// Add appends attributes to the event. These attributes will be included
// in the log entry when [End] is called. Prefer using the context-based
// [Set] function when possible, as it doesn't require passing the event directly.
func (e *Event) Add(attrs ...slog.Attr) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.attrs = append(e.attrs, attrs...)
}

// Set is an alias for [Event.Add], provided for consistency with the
// package-level [Set] function.
func (e *Event) Set(attrs ...slog.Attr) {
	e.Add(attrs...)
}

// SetError records an error on the event. Multiple errors can be recorded
// and will appear as "error-0", "error-1", etc. in the log output. Nil
// errors are silently ignored, so callers don't need to check before calling.
//
// Events with errors are logged at error level when [End] is called.
func (e *Event) SetError(err error) {
	if err == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = append(e.errors, err)
}

// fromContext retrieves the current event from the context. Returns false
// if no event exists, which happens when [Start] was never called on this
// context chain.
func fromContext(ctx context.Context) (*Event, bool) {
	event, ok := ctx.Value(eventKey{}).(*Event)
	return event, ok
}
