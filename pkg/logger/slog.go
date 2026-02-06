package logger

// Debug logs a message at debug level, bypassing the wide event system.
// Use for detailed diagnostic information during development. Arguments
// are key-value pairs in the same format as [slog.Logger.Debug].
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs a message at info level, bypassing the wide event system.
// Use for notable events during normal operation. Arguments are key-value
// pairs in the same format as [slog.Logger.Info].
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs a message at warn level, bypassing the wide event system.
// Use for unexpected situations that aren't errors but may indicate problems.
// Arguments are key-value pairs in the same format as [slog.Logger.Warn].
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs a message at error level, bypassing the wide event system.
// Use for failures that need attention. For request-scoped errors, prefer
// [SetError] to attach the error to the current wide event instead.
// Arguments are key-value pairs in the same format as [slog.Logger.Error].
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}
