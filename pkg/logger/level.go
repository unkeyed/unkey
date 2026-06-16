package logger

import (
	"context"
	"log/slog"
)

// minLevelHandler wraps a [slog.Handler] and drops records below min,
// regardless of the wrapped handler's own threshold. It lets a noisy
// dependency be held to a higher level than the rest of the app without
// touching the global level.
type minLevelHandler struct {
	inner slog.Handler
	min   slog.Level
}

var _ slog.Handler = minLevelHandler{inner: nil, min: 0}

// AtLevel returns a handler that forwards to h only for records at or above
// min, leaving h's formatting and sinks unchanged. Use it to quiet a chatty
// dependency — e.g. give the restate SDK an AtLevel(GetHandler(), LevelWarn)
// so its per-invocation INFO chatter is dropped while its warnings and errors
// still surface, and the app's own INFO/DEBUG logs are unaffected.
func AtLevel(h slog.Handler, min slog.Level) slog.Handler {
	return minLevelHandler{inner: h, min: min}
}

func (h minLevelHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.min && h.inner.Enabled(ctx, level)
}

func (h minLevelHandler) Handle(ctx context.Context, record slog.Record) error {
	if record.Level < h.min {
		return nil
	}
	return h.inner.Handle(ctx, record)
}

func (h minLevelHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return minLevelHandler{inner: h.inner.WithAttrs(attrs), min: h.min}
}

func (h minLevelHandler) WithGroup(name string) slog.Handler {
	return minLevelHandler{inner: h.inner.WithGroup(name), min: h.min}
}
