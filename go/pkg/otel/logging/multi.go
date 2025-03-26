package logging

import (
	"context"
	"errors"
	"log/slog"
)

type MultiHandler struct {
	Handlers []slog.Handler
}

var _ slog.Handler = (*MultiHandler)(nil)

func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.Handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

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

func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {

	newHandlers := make([]slog.Handler, len(h.Handlers))
	for i := range h.Handlers {
		newHandlers[i] = h.Handlers[i].WithAttrs(attrs)
	}
	return &MultiHandler{newHandlers}

}

func (h *MultiHandler) WithGroup(name string) slog.Handler {

	newHandlers := make([]slog.Handler, len(h.Handlers))
	for i := range h.Handlers {
		newHandlers[i] = h.Handlers[i].WithGroup(name)
	}
	return &MultiHandler{newHandlers}

}
