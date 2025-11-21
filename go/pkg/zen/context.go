package zen

import "context"

// ContextKey provides type-safe context storage using generics.
// It eliminates the need for type assertions and provides compile-time
// type safety when storing and retrieving values from context.
//
// Example usage:
//
//	var userIDKey = zen.NewContextKey[string]("user_id")
//	ctx = userIDKey.WithValue(ctx, "user123")
//	userID, ok := userIDKey.FromContext(ctx)
type ContextKey[T any] struct {
	name string
}

// NewContextKey creates a new typed context key with the given name.
// The name is used for debugging and doesn't need to be globally unique
// since the key itself is used as the context key.
func NewContextKey[T any](name string) ContextKey[T] {
	return ContextKey[T]{name: name}
}

// WithValue stores a value in the context using this key.
// Returns a new context with the value stored.
func (k ContextKey[T]) WithValue(ctx context.Context, value T) context.Context {
	return context.WithValue(ctx, k, value)
}

// FromContext retrieves a value from the context using this key.
// Returns the value and true if found, or the zero value and false if not found.
func (k ContextKey[T]) FromContext(ctx context.Context) (T, bool) {
	value, ok := ctx.Value(k).(T)
	return value, ok
}

// sessionKey is the context key for storing the session pointer.
var sessionKey = NewContextKey[*Session]("session")

// WithSession stores a session pointer in the context, making it available
// to downstream handlers and packages for operations like adding headers.
//
// This function enables patterns where middleware or handlers need to modify
// HTTP response headers from within cache operations or other utility functions
// that don't have direct access to the session. The session is stored using a
// private key type to prevent conflicts with other context values.
//
// Parameters:
//   - ctx: The parent context to extend with session storage
//   - session: The zen session to store. Must not be nil.
//
// Returns a new context with the session stored. The original context is
// not modified. The session can be retrieved later using SessionFromContext.
//
// Usage example:
//
//	ctx = zen.WithSession(ctx, session)
//	// Now downstream code can access the session:
//	if s, ok := zen.SessionFromContext(ctx); ok {
//	    s.AddHeader("X-Custom", "value")
//	}
//
// This is commonly used in middleware that enables functionality like cache
// debug headers, where cache operations need to write response headers without
// requiring explicit session passing through all function calls.
func WithSession(ctx context.Context, session *Session) context.Context {
	return sessionKey.WithValue(ctx, session)
}

// SessionFromContext retrieves the session pointer stored by WithSession.
//
// This function allows utility packages and handlers to access the HTTP session
// for operations like adding response headers, reading request data, or accessing
// session metadata. The session is safely type-cast from the context value.
//
// Parameters:
//   - ctx: The context to search for a stored session
//
// Returns:
//   - session: The stored session pointer, or nil if no session was found
//   - ok: true if a session was found, false otherwise
//
// The boolean return follows Go conventions for optional values and allows
// callers to distinguish between "no session stored" and "session stored but nil".
// However, WithSession should never store a nil session in practice.
//
// Usage example:
//
//	session, ok := zen.SessionFromContext(ctx)
//	if !ok {
//	    // No session available - cache debug disabled
//	    return
//	}
//	session.AddHeader("X-Cache-Debug", "api_by_id:150μs:FRESH")
//
// Performance note: Context value lookup is O(depth) where depth is the number
// of nested context.WithValue calls. This is typically very fast (<100ns) for
// normal request contexts, but avoid calling this in tight loops.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	return sessionKey.FromContext(ctx)
}
