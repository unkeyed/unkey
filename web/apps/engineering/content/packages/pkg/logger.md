---
title: logger
description: "provides wide-event logging with tail sampling support"
---

Package logger provides wide-event logging with tail sampling support.

Traditional logging emits entries as they happen, scattering request context across many lines. Wide-event logging takes a different approach: it collects attributes throughout a request's lifecycle and emits a single, rich log entry at the end. This reduces log volume while preserving full context for debugging.

The package is designed for request-scoped logging where you want to accumulate data across multiple function calls without threading a logger through every layer.

### Key Types

\[Event] accumulates attributes and errors throughout a request. Create one with \[Start], add data with \[Set] or \[SetError], and emit it with \[End].

\[Sampler] controls which events are emitted. Use \[AlwaysSample] during development and \[TailSampler] in production to reduce volume while preserving errors and slow requests.

\[MultiHandler] fans out log records to multiple \[slog.Handler] implementations, enabling simultaneous output to console and structured backends.

### Usage

Start an event at the beginning of a request, add attributes as you process, and end it when done:

	ctx, event := logger.Start(ctx, slog.String("handler", "createKey"))
	defer logger.End(ctx)

	logger.Set(ctx, slog.String("userId", user.ID))
	if err != nil {
	    logger.SetError(ctx, err)
	    return err
	}

The event is stored in the context, so any function with access to ctx can add attributes without needing a reference to the event itself.

### Tail Sampling

Tail sampling decides whether to emit a log after t pgit puthe request completes, rather than at the start. This allows sampling decisions based on outcome: always log errors and slow requests, sample routine successes at a lower rate.

	logger.SetSampler(logger.TailSampler{
	    SlowThreshold: time.Second, // always log slow requests
	    SampleRate:    0.1,         // sample 10% of normal requests
	})

### Standalone Logging

For logs outside request context, use the standard level functions \[Debug], \[Info], \[Warn], and \[Error]. These bypass the wide-event system and log immediately.

### Configuration

Use \[AddHandler] to add additional log destinations, \[AddBaseAttrs] to include attributes in all logs, and \[SetSampler] to configure sampling behavior. The package defaults to \[slog.Default] with \[AlwaysSample].

## Variables

Package-level state for the global logger, sampler, and base attributes. Protected by mu for concurrent access during configuration.
```go
var (
	logger    *slog.Logger
	sampler   Sampler
	baseAttrs []slog.Attr
	mu        sync.Mutex
)
```

```go
var _ slog.Handler = (*MultiHandler)(nil)
```


## Functions

### func AddBaseAttrs

```go
func AddBaseAttrs(attrs ...slog.Attr)
```

AddBaseAttrs appends attributes that will be included in every log entry. Use this to add service-level context like version, environment, or instance ID.

Safe for concurrent use. Attributes are appended, not replaced, so multiple calls accumulate.

### func AddHandler

```go
func AddHandler(newHandler slog.Handler)
```

AddHandler registers an additional \[slog.Handler] to receive log records. Handlers are composed using \[MultiHandler], so each log entry is sent to all registered handlers. This enables simultaneous output to multiple destinations like console and a structured logging backend.

Safe for concurrent use. Handlers added after logging has started will receive all subsequent log entries.

### func Debug

```go
func Debug(msg string, args ...any)
```

Debug logs a message at debug level, bypassing the wide event system. Use for detailed diagnostic information during development. Arguments are key-value pairs in the same format as \[slog.Logger.Debug].

### func Error

```go
func Error(msg string, args ...any)
```

Error logs a message at error level, bypassing the wide event system. Use for failures that need attention. For request-scoped errors, prefer \[SetError] to attach the error to the current wide event instead. Arguments are key-value pairs in the same format as \[slog.Logger.Error].

### func GetHandler

```go
func GetHandler() slog.Handler
```

GetHandler returns the current \[slog.Handler] used by the global logger. This can be used to inspect or wrap the handler for custom behavior.

Safe for concurrent use.

### func Info

```go
func Info(msg string, args ...any)
```

Info logs a message at info level, bypassing the wide event system. Use for notable events during normal operation. Arguments are key-value pairs in the same format as \[slog.Logger.Info].

### func Set

```go
func Set(ctx context.Context, attrs ...slog.Attr)
```

Set adds attributes to the current event stored in the context. If no event exists (i.e., \[Start] was never called), the call is silently ignored. This allows logging code to work safely in both request and non-request contexts.

### func SetError

```go
func SetError(ctx context.Context, err error)
```

SetError records an error on the current event. Multiple errors can be recorded and will appear as "error-0", "error-1", etc. in the log output. If no event exists in the context, the call is silently ignored.

Events with errors are logged at error level when \[End] is called and receive priority sampling when using \[TailSampler].

### func SetSampler

```go
func SetSampler(s Sampler)
```

SetSampler configures the sampling strategy for wide events. The sampler is consulted when \[End] is called to decide whether to emit the log entry. Use \[AlwaysSample] for development and \[TailSampler] for production.

Safe for concurrent use. Changing the sampler affects all subsequent calls to \[End], but does not affect events already in progress.

### func Warn

```go
func Warn(msg string, args ...any)
```

Warn logs a message at warn level, bypassing the wide event system. Use for unexpected situations that aren't errors but may indicate problems. Arguments are key-value pairs in the same format as \[slog.Logger.Warn].


## Types

### type AlwaysSample

```go
type AlwaysSample struct{}
```

AlwaysSample emits every event unconditionally. Use during development or debugging when you need to see all log output. Not recommended for production due to log volume.

#### func (AlwaysSample) Sample

```go
func (AlwaysSample) Sample(*Event) bool
```

Sample always returns true, emitting every event.

### type Event

```go
type Event struct {
	mu       sync.Mutex
	start    time.Time
	duration time.Duration
	message  string
	attrs    []slog.Attr
	errors   []error
	written  bool
}
```

Event accumulates attributes and errors throughout a request lifecycle, emitting a single log entry when \[End] is called. This "wide event" pattern reduces log volume while preserving full context for debugging.

Safe for concurrent use. Multiple goroutines can call \[Event.Add] and \[Event.SetError] on the same event.

#### func StartWideEvent

```go
func StartWideEvent(ctx context.Context, message string, attrs ...slog.Attr) (context.Context, *Event)
```

StartWideEvent begins a new wide event and stores it in the returned context. The event records the start time for duration calculation and any initial attributes provided.

The returned context should be passed to subsequent functions so they can add attributes via \[Set]. Call \[End] when the operation completes to emit the log entry based on the configured \[Sampler].

Typical usage:

	ctx, _ := logger.StartWideEvent(ctx, "incoming http request", slog.String("operation", "createKey"))
	defer logger.End(ctx)

#### func (Event) Add

```go
func (e *Event) Add(attrs ...slog.Attr)
```

Add appends attributes to the event. These attributes will be included in the log entry when \[End] is called. Prefer using the context-based \[Set] function when possible, as it doesn't require passing the event directly.

#### func (Event) End

```go
func (e *Event) End()
```

End emits the accumulated event as a log entry if the configured \[Sampler] allows it. Events with errors are logged at error level; others at info level.

Safe to call multiple times; only the first call emits a log entry. Subsequent calls are no-ops. Typically called via defer immediately after \[StartWideEvent].

#### func (Event) Set

```go
func (e *Event) Set(attrs ...slog.Attr)
```

Set is an alias for \[Event.Add], provided for consistency with the package-level \[Set] function.

#### func (Event) SetError

```go
func (e *Event) SetError(err error)
```

SetError records an error on the event. Multiple errors can be recorded and will appear as "error-0", "error-1", etc. in the log output. Nil errors are silently ignored, so callers don't need to check before calling.

Events with errors are logged at error level when \[End] is called.

### type MultiHandler

```go
type MultiHandler struct {
	Handlers []slog.Handler
}
```

MultiHandler fans out log records to multiple \[slog.Handler] implementations. Use this to send logs to multiple destinations simultaneously, such as console output and a structured logging backend.

A record is handled by all handlers that are enabled for that record's level. Errors from individual handlers are collected and returned as a joined error.

#### func (MultiHandler) Enabled

```go
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool
```

Enabled returns true if any handler is enabled for the given level.

#### func (MultiHandler) Handle

```go
func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error
```

Handle passes the record to all handlers that are enabled for its level. Returns a joined error if any handlers fail; handlers that succeed are not affected by failures in other handlers.

#### func (MultiHandler) WithAttrs

```go
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler
```

WithAttrs returns a new MultiHandler with the given attributes added to all child handlers.

#### func (MultiHandler) WithGroup

```go
func (h *MultiHandler) WithGroup(name string) slog.Handler
```

WithGroup returns a new MultiHandler with the given group name added to all child handlers.

### type Sampler

```go
type Sampler interface {
	Sample(event *Event) bool
}
```

Sampler decides whether an event should be emitted when \[End] is called. Implementations can inspect the completed event to make sampling decisions based on outcome, enabling tail sampling strategies.

### type TailSampler

```go
type TailSampler struct {

	// SlowThreshold defines what duration qualifies as "slow". Events
	// exceeding this duration are always sampled.
	SlowThreshold time.Duration

	// SampleRate is the baseline probability for events that aren't errors
	// and aren't slow. Set to 0.1 to sample 10% of normal traffic.
	SampleRate float64
}
```

TailSampler provides probabilistic sampling with bias toward errors and slow requests. This is the recommended sampler for production: it reduces log volume for routine successes while ensuring errors and performance issues are always captured.

Sampling rates are probabilities between 0.0 (never) and 1.0 (always). Events are evaluated in priority order: errors first, then slow requests, then baseline rate. An event matching multiple criteria (e.g., slow and has errors) is evaluated against the first matching rate.

#### func (TailSampler) Sample

```go
func (s TailSampler) Sample(event *Event) bool
```

Sample returns true if the event should be emitted based on configured rates. A single random value is generated and compared against each rate in order, so an event that matches multiple criteria still only has one chance to be sampled.

