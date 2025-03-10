// Package logging provides a flexible structured logging interface that follows
// the patterns of Go's standard slog package while abstracting over concrete
// logging implementations.
//
// The interface offers two styles of logging methods:
//
//  1. Standard methods (Debug, Info, Warn, Error) - without context but with
//     simple key-value pairs as variadic arguments.
//  2. Context-aware methods (DebugContext, InfoContext, WarnContext, ErrorContext) -
//     with context and structured slog.Attr values.
//
// This pattern matches slog's API design, making it familiar to Go developers.
//
// Example without context (simple key-value pairs):
//
//	logger.Info("User registered",
//	    "user_id", userID,
//	    "email", email,
//	    "age", age,
//	)
//
// Example with context (structured attributes):
//
//	logger.InfoContext(ctx, "User registered",
//	    slog.String("user_id", userID),
//	    slog.String("email", email),
//	    slog.Int("age", age),
//	)
//
// The package defines a common Logger interface and provides implementations for
// different environments (development, production) as well as a no-op logger for
// testing.
//
// Key features:
// - API mirroring Go's standard slog package
// - Context-aware and context-free logging methods
// - Structured logging for consistent, parseable logs
// - Leveled logging with Debug, Info, Warn, and Error levels
// - Support for log enrichment through With and WithAttrs methods
//
// Example usage:
//
//	// Create a logger
//	logger := logging.New(logging.Config{
//	    Development: true,  // Use human-readable format in development
//	    NoColor: false,     // Use colors in development mode
//	})
//
//	// Create derived logger with added fields
//	userLogger := logger.With(
//	    "user_id", user.ID,
//	    "email", user.Email,
//	)
//
//	// Standard logging
//	userLogger.Info("Profile updated", "field", "avatar")
//
//	// Context-aware logging with structured attributes
//	userLogger.InfoContext(ctx, "Payment processed",
//	    slog.Int("amount", payment.Amount),
//	    slog.String("currency", payment.Currency),
//	)
package logging
