package auditlog

import (
	"context"

	"github.com/unkeyed/unkey/pkg/uid"
)

// correlationCtxKey scopes the per-request correlation ID stash so audit
// log Insert calls in nested helpers can pick it up without the caller
// having to thread it through every function signature.
type correlationCtxKey struct{}

// WithCorrelation returns a derived ctx that carries id as the correlation
// key for any audit log events emitted via auditlog Insert. Use this at
// the top of a handler that fans out into multiple Insert calls (e.g.
// v2/keys.createKey -> withPermissions -> withRoles) so the dashboard can
// drill from any one event to the rest.
//
// Auto-batching covers the easier case (one Insert call with a slice of
// >1 events). This helper is for the harder case where the events are
// spread across separate Insert calls in different functions.
func WithCorrelation(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationCtxKey{}, id)
}

// CorrelationFrom returns the correlation ID stashed by WithCorrelation
// or the empty string if none has been set. The audit log Insert service
// reads this when an event in the batch has no explicit CorrelationID.
func CorrelationFrom(ctx context.Context) string {
	if id, ok := ctx.Value(correlationCtxKey{}).(string); ok {
		return id
	}
	return ""
}

// NewCorrelationID mints a fresh opaque correlation ID. Use at the top of
// a multi-Insert flow:
//
//	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
func NewCorrelationID() string {
	return uid.New(uid.CorrelationPrefix)
}
