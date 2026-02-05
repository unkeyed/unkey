package log

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Event accumulates attributes and errors throughout a request lifecycle.
// Safe for concurrent use.
type Event struct {
	mu       sync.Mutex
	start    time.Time
	duration time.Duration
	attrs    []slog.Attr
	errors   []error
	written  bool
}

// Add appends attributes to the event.
func (e *Event) Add(attrs ...slog.Attr) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.attrs = append(e.attrs, attrs...)
}

// Set is an alias for [Event.Add].
func (e *Event) Set(attrs ...slog.Attr) {
	e.Add(attrs...)
}

// SetError records an error on the event. Nil errors are ignored.
func (e *Event) SetError(err error) {
	if err == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	e.errors = append(e.errors, err)
}

type eventKey struct{}

func fromContext(ctx context.Context) (*Event, bool) {
	event, ok := ctx.Value(eventKey{}).(*Event)
	return event, ok
}
