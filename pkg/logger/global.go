package logger

import (
	"log/slog"
	"os"
	"strings"
	"sync"
)

// Package-level state for the global logger, sampler, and base attributes.
// Protected by mu for concurrent access during configuration.
//
// innerHandler holds the actual sink composition (text handler, plus any
// handlers added via [AddHandler], plus any base attrs). The exported
// `logger` is always `innerHandler` wrapped in [faultHandler] so error
// enrichment runs once at the top of the stack and the enriched record
// fans out to every sink.
var (
	logger       *slog.Logger
	innerHandler slog.Handler
	sampler      Sampler
	mu           sync.Mutex
)

func init() {
	mu = sync.Mutex{}
	// Minimum level from UNKEY_LOG_LEVEL (debug | info | warn | error),
	// defaulting to info. slog.Default()'s handler silently ignores
	// Debug regardless of env, so we wrap the stdlib TextHandler with
	// HandlerOptions{Level: ...} and route slog.Default() through it too
	// — that way plain `slog.Debug(...)` calls anywhere in the codebase
	// honor the same level as `logger.Debug(...)`.
	innerHandler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{ //nolint:exhaustruct // ReplaceAttr default
		Level:     levelFromEnv(),
		AddSource: true,
	})
	rebuild()
	sampler = AlwaysSample{}
}

// rebuild reinstalls the global logger and slog.Default() from the current
// innerHandler. Must be called with mu held.
func rebuild() {
	logger = slog.New(&faultHandler{inner: innerHandler})
	// slog.SetDefault captures the *Logger instance, not the package-level
	// variable. Re-apply so plain `slog.Info(...)` callers anywhere in the
	// codebase pick up handler/attr changes too.
	slog.SetDefault(logger)
}

// levelFromEnv parses UNKEY_LOG_LEVEL (case-insensitive: debug | info |
// warn | error). Anything unrecognized falls back to info, matching
// slog's stdlib default. Having this as an env var means flipping Debug
// on in prod is a DaemonSet patch, not a rebuild.
func levelFromEnv() slog.Level {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("UNKEY_LOG_LEVEL")))
	switch raw {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
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
	innerHandler = &MultiHandler{[]slog.Handler{innerHandler, newHandler}}
	rebuild()
}

// AddBaseAttrs appends attributes that will be included in every log entry.
// Use this to add service-level context like version, environment, or instance ID.
//
// Safe for concurrent use. Attributes are appended, not replaced, so multiple
// calls accumulate.
func AddBaseAttrs(attrs ...slog.Attr) {
	mu.Lock()
	defer mu.Unlock()
	innerHandler = innerHandler.WithAttrs(attrs)
	rebuild()
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
