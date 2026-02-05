// Package log provides wide-event logging with tail sampling support.
//
// Wide-event logging collects attributes throughout a request's lifecycle and
// emits a single log entry at the end. This reduces log volume while preserving
// full context for debugging. The package is designed for request-scoped logging
// where you want to accumulate data across multiple function calls.
//
// # Usage
//
// Start an event at the beginning of a request, add attributes as you process,
// and end it when done:
//
//	ctx := log.Start(ctx, slog.String("handler", "createKey"))
//	defer log.End(ctx)
//
//	log.Set(ctx, slog.String("userId", user.ID))
//	if err != nil {
//	    log.Error(ctx, err)
//	    return err
//	}
//
// # Sampling
//
// The package supports tail sampling through the [Sampler] interface. Tail sampling
// decides whether to emit a log after the request completes, allowing you to always
// log errors and slow requests while sampling routine successes. Use [TailSampler]
// for production or [AlwaysSample] for development.
//
// # Global Logger
//
// The package uses a global [Logger] configured via [SetLogger]. The default logger
// writes to slog.Default with no sampling.
package log
