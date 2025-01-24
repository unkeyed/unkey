package logging

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

type Config struct {
	Development bool
	NoColor     bool
}

type logger struct {
	logger *slog.Logger
}

func New(cfg Config) Logger {

	var handler slog.Handler
	switch {
	case cfg.Development && !cfg.NoColor:
		handler = tint.NewHandler(os.Stdout, &tint.Options{
			AddSource:   true,
			Level:       slog.LevelInfo,
			ReplaceAttr: nil,
			TimeFormat:  time.StampMilli,
			NoColor:     false,
		})
	case cfg.Development:
		handler = slog.NewTextHandler(os.Stdout, nil)
	default:
		handler = slog.NewJSONHandler(os.Stdout, nil)
	}

	l := slog.New(handler)
	return &logger{
		logger: l,
	}
}

func (l *logger) With(attrs ...slog.Attr) Logger {
	anys := make([]any, len(attrs))
	for i, a := range attrs {
		anys[i] = a
	}

	return &logger{
		logger: l.logger.With(anys...),
	}

}

func (l *logger) Debug(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelDebug, message, attrs...)
}
func (l *logger) Info(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, message, attrs...)

}
func (l *logger) Warn(ctx context.Context, message string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelWarn, message, attrs...)

}
func (l *logger) Error(ctx context.Context, message string, attrs ...slog.Attr) {

	l.logger.LogAttrs(ctx, slog.LevelError, message, attrs...)

}
