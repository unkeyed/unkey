// Package wide provides "one event per operation" wide event logging.
//
// The wide package implements a structured approach to logging where all
// context accumulated during processing is emitted as a single comprehensive
// event at the end. This makes debugging distributed systems easier
// by providing complete context in one log entry.
//
// Basic usage:
//
//	// At operation start
//	ctx, ev := wide.WithEventContext(ctx, wide.EventConfig{
//	    Logger:  logger,
//	    Sampler: sampler,
//	})
//	ev.Set("request_id", requestID)
//
//	// During processing
//	wide.Set(ctx, "key_id", keyID)
//	wide.Set(ctx, "cache_hit", true)
//
//	// At operation end - emit single wide event
//	ev.Emit()
package wide

import (
	"sync"
	"time"
)

// Logger is the interface for emitting log events.
// This matches the common structured logging pattern.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// EventConfig holds configuration for creating an EventContext.
type EventConfig struct {
	// Name is the event name used in the log message.
	// For HTTP services, typically "METHOD /path" (e.g., "POST /v2/keys.verifyKey").
	// Defaults to "event" if empty.
	Name string

	// Logger is where the event will be emitted.
	Logger Logger

	// Sampler controls whether events get logged.
	// If nil, all events are logged.
	Sampler Sampler
}

// EventContext accumulates fields throughout a request's lifecycle for emission
// as a single wide event at the end. It is safe for concurrent use.
type EventContext struct {
	mu       sync.RWMutex
	fields   map[string]any
	hasError bool
	start    time.Time
	name     string
	logger   Logger
	sampler  Sampler
}

// NewEventContext creates a new EventContext with the given configuration.
func NewEventContext(config EventConfig) *EventContext {
	name := config.Name
	if name == "" {
		name = "event"
	}
	return &EventContext{
		mu:       sync.RWMutex{},
		fields:   make(map[string]any),
		hasError: false,
		start:    time.Now(),
		name:     name,
		logger:   config.Logger,
		sampler:  config.Sampler,
	}
}

// Set adds or updates a single field in the event context.
// Keys should use snake_case (e.g., "request_id", "workspace_id").
// This method is safe for concurrent use.
func (e *EventContext) Set(key string, value any) {
	e.mu.Lock()
	e.fields[key] = value
	e.mu.Unlock()
}

// SetMany adds or updates multiple fields at once.
// This is more efficient than calling Set multiple times when adding
// several fields together.
func (e *EventContext) SetMany(fields map[string]any) {
	e.mu.Lock()
	for k, v := range fields {
		e.fields[k] = v
	}
	e.mu.Unlock()
}

// Get retrieves a field value by key. Returns nil and false if not found.
func (e *EventContext) Get(key string) (any, bool) {
	e.mu.RLock()
	val, ok := e.fields[key]
	e.mu.RUnlock()
	return val, ok
}

// MarkError marks this event as containing an error.
// Error events are always sampled (never dropped).
func (e *EventContext) MarkError() {
	e.mu.Lock()
	e.hasError = true
	e.mu.Unlock()
}

// HasError returns true if MarkError was called.
func (e *EventContext) HasError() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.hasError
}

// DurationMs returns the duration since the EventContext was created in milliseconds.
func (e *EventContext) DurationMs() int64 {
	return time.Since(e.start).Milliseconds()
}

// Duration returns the duration since the EventContext was created.
func (e *EventContext) Duration() time.Duration {
	return time.Since(e.start)
}

// StartTime returns the time when this EventContext was created.
func (e *EventContext) StartTime() time.Time {
	return e.start
}

// Fields returns all accumulated fields as a slice suitable for structured logging.
// The returned slice alternates between keys and values: [key1, val1, key2, val2, ...].
// This format is compatible with slog-style loggers.
//
// Example:
//
//	logger.Info("request", ev.Fields()...)
func (e *EventContext) Fields() []any {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make([]any, 0, len(e.fields)*2)
	for k, v := range e.fields {
		result = append(result, k, v)
	}
	return result
}

// FieldsMap returns a copy of all accumulated fields as a map.
// Useful for JSON encoding or inspection.
func (e *EventContext) FieldsMap() map[string]any {
	e.mu.RLock()
	defer e.mu.RUnlock()

	result := make(map[string]any, len(e.fields))
	for k, v := range e.fields {
		result[k] = v
	}
	return result
}

// FieldCount returns the number of fields accumulated.
func (e *EventContext) FieldCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.fields)
}

// Emit checks the sampler and emits the event if it should be logged.
// Returns true if the event was emitted, false if it was dropped by sampling.
//
// The sampler inspects the EventContext to decide (e.g., hasError, duration,
// status_code field for HTTP services).
//
// If no logger was configured, this is a no-op and returns false.
// If no sampler was configured, the event is always emitted.
func (e *EventContext) Emit() bool {
	if e.logger == nil {
		return false
	}

	// Check sampling decision
	if e.sampler != nil {
		decision := e.sampler.ShouldSample(e)
		if !decision.Sample {
			return false
		}
		e.Set(FieldSampleReason, decision.Reason)
	}

	// Emit the wide event at appropriate level
	if e.hasError {
		e.logger.Error(e.name, e.Fields()...)
	} else {
		e.logger.Info(e.name, e.Fields()...)
	}
	return true
}
