package logger

import (
	"context"
	"log/slog"

	"github.com/unkeyed/unkey/pkg/fault"
)

// faultHandler decorates a [slog.Handler] so that any [error] value
// attached to a log record is expanded into its full fault wrap chain.
//
// Two attributes are added when a fault-wrapped error is found:
//   - error.steps: the flattened chain returned by [fault.Flatten], with
//     the source file:line of every [fault.New]/[fault.Wrap] call.
//   - error.location: the file:line of the outermost wrap, which is
//     almost always what you want to grep for first.
//
// Doing this in a handler — instead of a wrapper function around slog —
// lets every emission path benefit (package-level slog calls, the wide-
// event system, third-party libraries logging via slog.Default()) and
// leaves stdlib slog to capture source PCs at the real call site.
type faultHandler struct {
	inner slog.Handler
}

var _ slog.Handler = (*faultHandler)(nil)

func (h *faultHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.inner.Enabled(ctx, l)
}

func (h *faultHandler) Handle(ctx context.Context, r slog.Record) error {
	// Walk the record's attrs once. The first error we find that carries
	// a fault chain wins; if a call site logs multiple errors, emitting
	// step groups for each would just create noise.
	var steps []fault.Step
	r.Attrs(func(a slog.Attr) bool {
		if err, ok := a.Value.Any().(error); ok {
			if s := fault.Flatten(err); len(s) > 0 {
				steps = s
				return false
			}
		}
		return true
	})

	if len(steps) > 0 {
		r.AddAttrs(
			slog.Any("error.steps", steps),
			slog.String("error.location", steps[len(steps)-1].Location),
		)
	}

	return h.inner.Handle(ctx, r)
}

func (h *faultHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &faultHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *faultHandler) WithGroup(name string) slog.Handler {
	return &faultHandler{inner: h.inner.WithGroup(name)}
}
