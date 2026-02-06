package logger

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// StartWideEvent begins a new wide event and stores it in the returned context.
// The event records the start time for duration calculation and any initial
// attributes provided.
//
// The returned context should be passed to subsequent functions so they can
// add attributes via [Set]. Call [End] when the operation completes to emit
// the log entry based on the configured [Sampler].
//
// Typical usage:
//
//	ctx, event := logger.StartWideEvent(ctx, "incoming http request", slog.String("operation", "createKey"))
//	defer event.End()
func StartWideEvent(ctx context.Context, message string, attrs ...slog.Attr) (context.Context, *Event) {
	now := time.Now()
	event := &Event{
		mu:       sync.Mutex{},
		start:    now,
		duration: 0,
		errors:   []error{},
		message:  message,
		written:  false,
		attrs:    attrs,
	}
	return context.WithValue(ctx, eventKey{}, event), event
}

// Set adds attributes to the current event stored in the context. If no event
// exists (i.e., [Start] was never called), the call is silently ignored. This
// allows logging code to work safely in both request and non-request contexts.
func Set(ctx context.Context, attrs ...slog.Attr) {
	event, ok := ctx.Value(eventKey{}).(*Event)
	if !ok {
		return
	}
	event.Add(attrs...)
}

// SetError records an error on the current event. Multiple errors can be
// recorded and will appear as "error-0", "error-1", etc. in the log output.
// If no event exists in the context, the call is silently ignored.
//
// Events with errors are logged at error level when [End] is called and
// receive priority sampling when using [TailSampler].
func SetError(ctx context.Context, err error) {

	event, ok := ctx.Value(eventKey{}).(*Event)
	if !ok {
		return
	}
	event.SetError(err)
}
