package logging

import (
	"context"
	"fmt"
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

// Handler returns the current slog.Handler used by the logging package.
// This is useful for integrating with libraries that accept a slog.Handler.
func Handler() slog.Handler {
	return handler
}

// logSkip logs with correct source location by skipping wrapper frames
func (l *logger) logSkip(ctx context.Context, level slog.Level, msg string, args ...any) {
	if !l.logger.Enabled(ctx, level) {
		return
	}
	var pcs [1]uintptr
	runtime.Callers(l.callDepth, pcs[:]) // skip [runtime.Callers, logSkip, wrapper]
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.Add(args...)
	if err := l.logger.Handler().Handle(ctx, r); err != nil {
		fmt.Fprintf(os.Stderr, "logging handler error: %v\n", err)
	}
}

// logger implements the Logger interface using Go's standard slog package.
type logger struct {
	logger    *slog.Logger
	callDepth int
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
		logger:    l,
		callDepth: 3,
	}
}

// With creates a new logger with the given key-value pairs always attached.
func (l *logger) With(args ...any) Logger {
	return &logger{
		logger:    l.logger.With(args...),
		callDepth: l.callDepth,
	}
}

// WithAttrs creates a new logger with the given attributes always attached.
func (l *logger) WithAttrs(attrs ...slog.Attr) Logger {
	anys := make([]any, len(attrs))
	for i, a := range attrs {
		anys[i] = a
	}

	return &logger{
		logger:    l.logger.With(anys...),
		callDepth: l.callDepth,
	}
}

// WithCallDepth creates a new logger with the given call depth always attached.
func (l *logger) WithCallDepth(depth int) Logger {

	return &logger{
		logger:    l.logger,
		callDepth: depth,
	}
}

// ---- Standard logging methods ----

// Debug logs a message at debug level with key-value pairs.
func (l *logger) Debug(msg string, args ...any) {
	l.logSkip(context.Background(), slog.LevelDebug, msg, args...)
}

// Info logs a message at info level with key-value pairs.
func (l *logger) Info(msg string, args ...any) {
	l.logSkip(context.Background(), slog.LevelInfo, msg, args...)
}

// Warn logs a message at warn level with key-value pairs.
func (l *logger) Warn(msg string, args ...any) {
	l.logSkip(context.Background(), slog.LevelWarn, msg, args...)
}

// Error logs a message at error level with key-value pairs.
func (l *logger) Error(msg string, args ...any) {
	l.logSkip(context.Background(), slog.LevelError, msg, args...)
}
