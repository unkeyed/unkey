package logger

import "log/slog"

// Debug, Info, Warn, and Error are direct aliases for the stdlib slog
// package-level functions. They bypass the wide event system and log
// immediately via [slog.Default], which is kept in sync with the global
// logger configured by [AddHandler] / [AddBaseAttrs].
//
// Aliasing (rather than wrapping in our own functions) is deliberate: it
// preserves the caller's program counter so [slog.HandlerOptions.AddSource]
// reports the real call site. A hand-written wrapper would add a frame and
// the source attribute would always point back at this file.
//
// Fault-wrapped errors passed in args are automatically expanded into an
// error.steps / error.location attribute pair by [faultHandler]; callers
// don't need to think about it. For request-scoped errors, prefer
// [SetError] to attach the error to the current wide event instead.
var (
	Debug = slog.Debug
	Info  = slog.Info
	Warn  = slog.Warn
	Error = slog.Error
)
