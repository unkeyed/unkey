package logging

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var handler slog.Handler

func init() {
	handler = tint.NewHandler(os.Stdout, &tint.Options{
		AddSource:   false,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
		TimeFormat:  time.StampMilli,
		NoColor:     false,
	})
}
func SetHandler(h slog.Handler) {
	handler = h
}

// logger implements the Logger interface using Go's standard slog package.
type logger struct {
	logger *slog.Logger
}

// New creates a new logger with the specified configuration.
// The logger adapts its format based on the environment:
// - In development mode, it uses a human-readable format with optional colors
// - In production mode, it uses structured JSON format
//
// Example:
//
//	// Development logger with colors
//	devLogger := logging.New(logging.Config{
//	    Development: true,
//	    NoColor: false,
//	})
//
//	// Production JSON logger
//	prodLogger := logging.New(logging.Config{
//	    Development: false,
//	})
func New() Logger {

	l := slog.New(handler)

	return &logger{
		logger: l,
	}
}

// With creates a new logger with the given key-value pairs always attached.
func (l *logger) With(args ...any) Logger {
	return &logger{
		logger: l.logger.With(args...),
	}
}

// WithAttrs creates a new logger with the given attributes always attached.
func (l *logger) WithAttrs(attrs ...slog.Attr) Logger {
	anys := make([]any, len(attrs))
	for i, a := range attrs {
		anys[i] = a
	}

	return &logger{
		logger: l.logger.With(anys...),
	}
}

// ---- Standard logging methods ----

// Debug logs a message at debug level with key-value pairs.
func (l *logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

// Info logs a message at info level with key-value pairs.
func (l *logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

// Warn logs a message at warn level with key-value pairs.
func (l *logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

// Error logs a message at error level with key-value pairs.
func (l *logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

// ---- Context-aware logging methods ----

// DebugContext logs a message at debug level with context and structured attributes.
func (l *logger) DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelDebug, msg, attrs...)
}

// InfoContext logs a message at info level with context and structured attributes.
func (l *logger) InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelInfo, msg, attrs...)
}

// WarnContext logs a message at warn level with context and structured attributes.
func (l *logger) WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelWarn, msg, attrs...)
}

// ErrorContext logs a message at error level with context and structured attributes.
func (l *logger) ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	l.logger.LogAttrs(ctx, slog.LevelError, msg, attrs...)
}
