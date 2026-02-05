package logger

import (
	"log/slog"
	"sync"
)

// Package-level state for the global logger, sampler, and base attributes.
// Protected by mu for concurrent access during configuration.
var (
	logger    *slog.Logger
	sampler   Sampler
	baseAttrs []slog.Attr
	mu        sync.Mutex
)

func init() {
	mu = sync.Mutex{}
	logger = slog.Default()
	sampler = AlwaysSample{}
	baseAttrs = []slog.Attr{}
}

// GetHandler returns the current [slog.Handler] used by the global logger.
// This can be used to inspect or wrap the handler for custom behavior.
//
// Safe for concurrent use.
func GetHandler() slog.Handler {
	mu.Lock()
	defer mu.Unlock()
	return logger.Handler()
}

// AddHandler registers an additional [slog.Handler] to receive log records.
// Handlers are composed using [MultiHandler], so each log entry is sent to
// all registered handlers. This enables simultaneous output to multiple
// destinations like console and a structured logging backend.
//
// Safe for concurrent use. Handlers added after logging has started will
// receive all subsequent log entries.
func AddHandler(newHandler slog.Handler) {
	mu.Lock()
	defer mu.Unlock()
	logger = slog.New(&MultiHandler{[]slog.Handler{logger.Handler(), newHandler}})
}

// AddBaseAttrs appends attributes that will be included in every log entry.
// Use this to add service-level context like version, environment, or instance ID.
//
// Safe for concurrent use. Attributes are appended, not replaced, so multiple
// calls accumulate.
func AddBaseAttrs(attrs ...slog.Attr) {
	mu.Lock()
	defer mu.Unlock()
	baseAttrs = append(baseAttrs, attrs...)
}

// SetSampler configures the sampling strategy for wide events. The sampler
// is consulted when [End] is called to decide whether to emit the log entry.
// Use [AlwaysSample] for development and [TailSampler] for production.
//
// Safe for concurrent use. Changing the sampler affects all subsequent calls
// to [End], but does not affect events already in progress.
func SetSampler(s Sampler) {
	mu.Lock()
	defer mu.Unlock()
	sampler = s
}
