package logging

import (
	"context"
	"log/slog"
)

type Logger interface {
	With(attrs ...slog.Attr) Logger

	Debug(ctx context.Context, message string, attrs ...slog.Attr)
	Info(ctx context.Context, message string, attrs ...slog.Attr)
	Warn(ctx context.Context, message string, attrs ...slog.Attr)
	Error(ctx context.Context, message string, attrs ...slog.Attr)
}
