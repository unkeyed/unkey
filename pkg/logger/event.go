package logger

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
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
	message  string
	attrs    []slog.Attr
	errors   []error
	written  bool
	// pc is the program counter captured at [StartWideEvent] so the emitted
	// log record's source attribute points at the caller that opened the
	// event, not at this file. Zero means no PC was captured.
	pc uintptr
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

// End emits the accumulated event as a log entry if the configured [Sampler]
// allows it. Events with errors are logged at error level; others at info level.
//
// Safe to call multiple times; only the first call emits a log entry. Subsequent
// calls are no-ops. Typically called via defer immediately after [StartWideEvent].
func (e *Event) End() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.written {
		return
	}
	e.written = true

	if !sampler.Sample(e) {
		return
	}

	errors := []slog.Attr{}
	for i, err := range e.errors {
		code, _ := fault.GetCode(err)
		errors = append(errors, slog.Group(fmt.Sprintf("%d", i),
			slog.String("internal", fault.InternalMessage(err)),
			slog.String("public", fault.UserFacingMessage(err)),
			slog.String("code", string(code)),
			slog.Any("steps", fault.Flatten(err)),
		))
	}

	attrs := make([]slog.Attr, 0, len(e.attrs)+2)
	attrs = append(attrs,
		slog.GroupAttrs("errors", errors...),
		slog.GroupAttrs("log_meta",
			slog.Time("start", e.start),
			slog.Duration("duration", time.Since(e.start)),
		),
	)
	attrs = append(attrs, e.attrs...)

	level := slog.LevelInfo
	msg := e.message
	if len(e.errors) > 0 {
		level = slog.LevelError
		msg = "error"
	}

	ctx := context.Background()
	if !logger.Enabled(ctx, level) {
		return
	}

	// Build the record manually so the source attribute points at the
	// caller of StartWideEvent (the handler/middleware that opened the
	// event), not at this file. Using logger.Error/Info here would capture
	// the PC of this line instead, which is useless for debugging.
	r := slog.NewRecord(time.Now(), level, msg, e.pc)
	r.AddAttrs(attrs...)
	_ = logger.Handler().Handle(ctx, r)
}
