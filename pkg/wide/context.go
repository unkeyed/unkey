package wide

import "context"

// contextKey is an unexported type for context keys to prevent collisions.
type contextKey struct{}

// eventContextKey is the context key for EventContext values.
var eventContextKey = contextKey{}

// WithEventContext creates a new EventContext with logger and sampler,
// and stores it in the context. The EventContext can later emit itself via Emit().
//
// Typical usage:
//
//	ctx, ev := wide.WithEventContext(ctx, wide.EventConfig{
//	    Logger:  logger,
//	    Sampler: sampler,
//	})
//	// ... do work ...
//	ev.Emit()
func WithEventContext(ctx context.Context, config EventConfig) (context.Context, *EventContext) {
	ev := NewEventContext(config)
	return context.WithValue(ctx, eventContextKey, ev), ev
}

// FromContext retrieves the EventContext from the context.
// Returns nil if no EventContext is present.
//
// Example:
//
//	if ev := wide.FromContext(ctx); ev != nil {
//	    ev.Set("key_id", keyID)
//	}
func FromContext(ctx context.Context) *EventContext {
	ev, _ := ctx.Value(eventContextKey).(*EventContext)
	return ev
}

// Set is a convenience function to set a field on the EventContext in the context.
// If no EventContext is present, this is a no-op.
//
// This is the primary way for handlers and services to add context during
// request processing:
//
//	wide.Set(ctx, "key_id", keyID)
//	wide.Set(ctx, "cache_hit", true)
func Set(ctx context.Context, key string, value any) {
	if ev := FromContext(ctx); ev != nil {
		ev.Set(key, value)
	}
}

// SetMany is a convenience function to set multiple fields at once.
// If no EventContext is present, this is a no-op.
//
// Example:
//
//	wide.SetMany(ctx, map[string]any{
//	    "key_id": keyID,
//	    "api_id": apiID,
//	    "cache_hit": true,
//	})
func SetMany(ctx context.Context, fields map[string]any) {
	if ev := FromContext(ctx); ev != nil {
		ev.SetMany(fields)
	}
}

// MarkError marks the EventContext as containing an error.
// Error events are always sampled (never dropped).
// If no EventContext is present, this is a no-op.
func MarkError(ctx context.Context) {
	if ev := FromContext(ctx); ev != nil {
		ev.MarkError()
	}
}

// Get retrieves a field value from the EventContext in the context.
// Returns nil and false if no EventContext is present or if the key is not found.
func Get(ctx context.Context, key string) (any, bool) {
	if ev := FromContext(ctx); ev != nil {
		return ev.Get(key)
	}
	return nil, false
}
