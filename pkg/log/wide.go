package log

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Logger wraps slog.Logger with sampling and wide-event support.
type Logger struct {
	logger    *slog.Logger
	sampler   Sampler
	clock     clock.Clock
	baseAttrs []slog.Attr
}

var (
	global *Logger
	mu     sync.Mutex
)

func init() {
	mu = sync.Mutex{}
	global = &Logger{
		logger:    slog.Default(),
		sampler:   AlwaysSample{},
		clock:     clock.New(),
		baseAttrs: []slog.Attr{},
	}
}

// SetLogger configures the global logger used by [Start], [Set], [Error], and [End].
// If logger is nil, the call is ignored.
func SetLogger(logger *Logger) {
	if logger == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	global = logger
}

// Start begins a new wide event and stores it in the returned context.
// Call [End] when the operation completes to emit the log entry.
func Start(ctx context.Context, attrs ...slog.Attr) context.Context {
	return context.WithValue(ctx, eventKey{}, &Event{
		mu:       sync.Mutex{},
		start:    global.clock.Now(),
		duration: 0,
		errors:   []error{},
		written:  false,
		attrs:    append([]slog.Attr{}, attrs...),
	})
}

// Set adds attributes to the current event. If no event exists in the context,
// the call is silently ignored.
func Set(ctx context.Context, attrs ...slog.Attr) {
	event, ok := fromContext(ctx)
	if !ok {
		return
	}
	event.Add(attrs...)
}

// Error records an error on the current event. Multiple errors can be recorded.
// If no event exists in the context, the call is silently ignored.
func Error(ctx context.Context, err error) {
	event, ok := fromContext(ctx)
	if !ok {
		return
	}
	event.SetError(err)
}

// End completes the event, applies sampling, and emits the log entry if sampled.
// Events with errors are logged at error level; successful events at info level.
// Calling End multiple times on the same event is safe; only the first call emits.
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

	if !global.sampler.Sample(e) {
		return
	}

	errors := []slog.Attr{}
	for i, err := range e.errors {

		errors = append(errors, slog.String(fmt.Sprintf("error-%d", i), fault.InternalMessage(err)))
	}

	casted := []any{
		slog.GroupAttrs("errors", errors...),
		slog.GroupAttrs("time",
			slog.Time("start", e.start),
			slog.Duration("duration", time.Since(e.start)),
		),
	}
	for _, attr := range e.attrs {
		casted = append(casted, attr)
	}

	if len(e.errors) > 0 {

		global.logger.Error("error", casted...)
	} else {
		global.logger.Info("wide event", casted...)
	}

}
