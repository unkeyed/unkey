package logging

import (
	"context"
	"log/slog"
)

type noop struct {
}

func NewNoop() Logger {
	return &noop{}
}

func (l *noop) With(attrs ...slog.Attr) Logger {

	return l

}

func (l *noop) Debug(ctx context.Context, message string, attrs ...slog.Attr) {

}
func (l *noop) Info(ctx context.Context, message string, attrs ...slog.Attr) {

}
func (l *noop) Warn(ctx context.Context, message string, attrs ...slog.Attr) {

}
func (l *noop) Error(ctx context.Context, message string, attrs ...slog.Attr) {
}
