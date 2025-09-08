package logging

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/lmittmann/tint"
)

var handler slog.Handler

func init() {
	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	handler = tint.NewHandler(os.Stdout, &tint.Options{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: nil,
		TimeFormat:  time.StampMilli,
		NoColor:     false,
	})
}

func AddHandler(h slog.Handler) {
	handler = &MultiHandler{[]slog.Handler{handler, h}}
}

func SetHandler(h slog.Handler) {
	handler = h
}

// logSkip logs with correct source location by skipping wrapper frames
func logSkip(l *slog.Logger, ctx context.Context, level slog.Level, msg string, args ...any) {
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [runtime.Callers, logSkip, wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	_ = l.Handler().Handle(ctx, r)
}

// logAttrs logs with attributes and correct source location
func logAttrs(l *slog.Logger, ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	if !l.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip [runtime.Callers, logAttrs, wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(attrs...)
	_ = l.Handler().Handle(ctx, r)
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
	logSkip(l.logger, context.Background(), slog.LevelDebug, msg, args...)
}

// Info logs a message at info level with key-value pairs.
func (l *logger) Info(msg string, args ...any) {
	logSkip(l.logger, context.Background(), slog.LevelInfo, msg, args...)
}

// Warn logs a message at warn level with key-value pairs.
func (l *logger) Warn(msg string, args ...any) {
	logSkip(l.logger, context.Background(), slog.LevelWarn, msg, args...)
}

// Error logs a message at error level with key-value pairs.
func (l *logger) Error(msg string, args ...any) {
	logSkip(l.logger, context.Background(), slog.LevelError, msg, args...)
}

// ---- Context-aware logging methods ----

// DebugContext logs a message at debug level with context and structured attributes.
func (l *logger) DebugContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(l.logger, ctx, slog.LevelDebug, msg, attrs...)
}

// InfoContext logs a message at info level with context and structured attributes.
func (l *logger) InfoContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(l.logger, ctx, slog.LevelInfo, msg, attrs...)
}

// WarnContext logs a message at warn level with context and structured attributes.
func (l *logger) WarnContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(l.logger, ctx, slog.LevelWarn, msg, attrs...)
}

// ErrorContext logs a message at error level with context and structured attributes.
func (l *logger) ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr) {
	logAttrs(l.logger, ctx, slog.LevelError, msg, attrs...)
}
