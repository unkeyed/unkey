// Package logger provides wide-event logging with tail sampling support.
//
// Traditional logging emits entries as they happen, scattering request context
// across many lines. Wide-event logging takes a different approach: it collects
// attributes throughout a request's lifecycle and emits a single, rich log entry
// at the end. This reduces log volume while preserving full context for debugging.
//
// The package is designed for request-scoped logging where you want to accumulate
// data across multiple function calls without threading a logger through every layer.
//
// # Key Types
//
// [Event] accumulates attributes and errors throughout a request. Create one with
// [Start], add data with [Set] or [SetError], and emit it with [End].
//
// [Sampler] controls which events are emitted. Use [AlwaysSample] during development
// and [TailSampler] in production to reduce volume while preserving errors and slow
// requests.
//
// [MultiHandler] fans out log records to multiple [slog.Handler] implementations,
// enabling simultaneous output to console and structured backends.
//
// # Usage
//
// Start an event at the beginning of a request, add attributes as you process,
// and end it when done:
//
//	ctx, event := logger.Start(ctx, slog.String("handler", "createKey"))
//	defer logger.End(ctx)
//
//	logger.Set(ctx, slog.String("userId", user.ID))
//	if err != nil {
//	    logger.SetError(ctx, err)
//	    return err
//	}
//
// The event is stored in the context, so any function with access to ctx can add
// attributes without needing a reference to the event itself.
//
// # Tail Sampling
//
// Tail sampling decides whether to emit a log after t pgit puthe request completes, rather
// than at the start. This allows sampling decisions based on outcome: always log
// errors and slow requests, sample routine successes at a lower rate.
//
//	logger.SetSampler(logger.TailSampler{
//	    SlowThreshold: time.Second, // always log slow requests
//	    SampleRate:    0.1,         // sample 10% of normal requests
//	})
//
// # Standalone Logging
//
// For logs outside request context, use the standard level functions [Debug], [Info],
// [Warn], and [Error]. These bypass the wide-event system and log immediately.
//
// # Configuration
//
// Use [AddHandler] to add additional log destinations, [AddBaseAttrs] to include
// attributes in all logs, and [SetSampler] to configure sampling behavior. The
// package defaults to [slog.Default] with [AlwaysSample].
package logger
