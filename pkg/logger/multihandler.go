package logger

import (
	"context"
	"errors"
	"log/slog"
)

// MultiHandler fans out log records to multiple [slog.Handler] implementations.
// Use this to send logs to multiple destinations simultaneously, such as
// console output and a structured logging backend.
//
// A record is handled by all handlers that are enabled for that record's level.
// Errors from individual handlers are collected and returned as a joined error.
type MultiHandler struct {
	Handlers []slog.Handler
}

var _ slog.Handler = (*MultiHandler)(nil)

// Enabled returns true if any handler is enabled for the given level.
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.Handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle passes the record to all handlers that are enabled for its level.
// Returns a joined error if any handlers fail; handlers that succeed are
// not affected by failures in other handlers.
func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	var errs []error
	for _, handler := range h.Handlers {
		if handler.Enabled(ctx, record.Level) {
			err := handler.Handle(ctx, record)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// WithAttrs returns a new MultiHandler with the given attributes added to
// all child handlers.
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.Handlers))
	for i := range h.Handlers {
		newHandlers[i] = h.Handlers[i].WithAttrs(attrs)
	}
	return &MultiHandler{newHandlers}
}

// WithGroup returns a new MultiHandler with the given group name added to
// all child handlers.
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.Handlers))
	for i := range h.Handlers {
		newHandlers[i] = h.Handlers[i].WithGroup(name)
	}
	return &MultiHandler{newHandlers}
}
