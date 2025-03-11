package logging

import (
	"context"
	"log/slog"
)

// Logger defines a structured logging interface that follows the patterns
// of Go's standard slog package, with methods for both context-aware and
// context-free logging.
type Logger interface {
	// With creates a new logger with the given attributes always attached
	// to future log entries. This is useful for adding context that should
	// appear in all subsequent logs.
	//
	// Example:
	//  requestLogger := logger.With(
	//      "request_id", requestID,
	//      "method", r.Method,
	//  )
	With(args ...any) Logger

	// WithAttrs creates a new logger with the given slog.Attr values always
	// attached to future log entries. This provides more type safety and
	// flexibility than the With method.
	//
	// Example:
	//  requestLogger := logger.WithAttrs(
	//      slog.String("request_id", requestID),
	//      slog.String("method", r.Method),
	//  )
	WithAttrs(attrs ...slog.Attr) Logger

	// ---- Standard logging methods (without context) ----

	// Debug logs a message at debug level with simple key-value pairs.
	// Keys must be strings.
	//
	// Example:
	//  logger.Debug("Processing request payload",
	//      "size", len(payload),
	//      "content_type", contentType,
	//  )
	Debug(msg string, args ...any)

	// Info logs a message at info level with simple key-value pairs.
	//
	// Example:
	//  logger.Info("User authenticated",
	//      "user_id", user.ID,
	//      "auth_method", "password",
	//  )
	Info(msg string, args ...any)

	// Warn logs a message at warn level with simple key-value pairs.
	//
	// Example:
	//  logger.Warn("API rate limit approaching threshold",
	//      "client_id", clientID,
	//      "requests", count,
	//      "limit", limit,
	//  )
	Warn(msg string, args ...any)

	// Error logs a message at error level with simple key-value pairs.
	//
	// Example:
	//  logger.Error("Failed to process payment",
	//      "payment_id", paymentID,
	//      "error", err.Error(),
	//  )
	Error(msg string, args ...any)

	// ---- Context-aware logging methods ----

	// DebugContext logs a message at debug level with the given context
	// and structured attributes.
	//
	// Example:
	//  logger.DebugContext(ctx, "Processing request payload",
	//      slog.Int("size", len(payload)),
	//      slog.String("content_type", contentType),
	//  )
	DebugContext(ctx context.Context, msg string, attrs ...slog.Attr)

	// InfoContext logs a message at info level with the given context
	// and structured attributes.
	//
	// Example:
	//  logger.InfoContext(ctx, "User authenticated",
	//      slog.String("user_id", user.ID),
	//      slog.String("auth_method", "password"),
	//  )
	InfoContext(ctx context.Context, msg string, attrs ...slog.Attr)

	// WarnContext logs a message at warn level with the given context
	// and structured attributes.
	//
	// Example:
	//  logger.WarnContext(ctx, "API rate limit approaching threshold",
	//      slog.String("client_id", clientID),
	//      slog.Int("requests", count),
	//      slog.Int("limit", limit),
	//  )
	WarnContext(ctx context.Context, msg string, attrs ...slog.Attr)

	// ErrorContext logs a message at error level with the given context
	// and structured attributes.
	//
	// Example:
	//  logger.ErrorContext(ctx, "Failed to process payment",
	//      slog.String("payment_id", paymentID),
	//      slog.String("error", err.Error()),
	//  )
	ErrorContext(ctx context.Context, msg string, attrs ...slog.Attr)
}
