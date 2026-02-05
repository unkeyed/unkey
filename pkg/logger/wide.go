package logger

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Start begins a new wide event and stores it in the returned context.
// The event records the start time for duration calculation and any initial
// attributes provided.
//
// The returned context should be passed to subsequent functions so they can
// add attributes via [Set]. Call [End] when the operation completes to emit
// the log entry based on the configured [Sampler].
//
// Typical usage:
//
//	ctx, _ := logger.Start(ctx, slog.String("operation", "createKey"))
//	defer logger.End(ctx)
func Start(ctx context.Context, attrs ...slog.Attr) (context.Context, *Event) {
	now := time.Now()
	event := &Event{
		mu:       sync.Mutex{},
		start:    now,
		duration: 0,
		errors:   []error{},
		written:  false,
		attrs:    attrs,
	}
	return context.WithValue(ctx, eventKey{}, event), event
}

// Set adds attributes to the current event stored in the context. If no event
// exists (i.e., [Start] was never called), the call is silently ignored. This
// allows logging code to work safely in both request and non-request contexts.
func Set(ctx context.Context, attrs ...slog.Attr) {
	event, ok := fromContext(ctx)
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
	event, ok := fromContext(ctx)
	if !ok {
		return
	}
	event.SetError(err)
}

// End completes the event, applies sampling, and emits the log entry if sampled.
// The log entry includes all accumulated attributes, recorded errors, start time,
// and total duration.
//
// Events with errors are logged at error level; successful events at info level.
// Calling End multiple times on the same event is safe; only the first call emits.
// If no event exists in the context, the call is silently ignored.
func End(ctx context.Context) {
	e, ok := fromContext(ctx)
	if !ok {
		return
	}

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

	casted := []any{
		slog.GroupAttrs("errors", errors...),
		slog.GroupAttrs("log_meta",
			slog.Time("start", e.start),
			slog.Duration("duration", time.Since(e.start)),
		),
	}
	for _, attr := range e.attrs {
		casted = append(casted, attr)
	}

	if len(e.errors) > 0 {
		logger.Error("error", casted...)
	} else {
		logger.Info("wide event", casted...)
	}

}
