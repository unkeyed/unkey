package ctxutil

import "context"

// contextKey is a type used to create unique keys for context values.
// Using a custom type prevents collisions with keys from other packages.
type contextKey string

const (
	// request_id is the context key for storing request identifiers
	request_id contextKey = "request_id"
)

// getValue returns the value for the given key from the context or its zero
// value if it doesn't exist. This provides type-safe access to context values.
//
// Example:
//
//	userID := getValue[string](ctx, userIDKey)
//	count := getValue[int](ctx, countKey) // Returns 0 if not set
func getValue[T any](ctx context.Context, key contextKey) T {
	val, ok := ctx.Value(key).(T)
	if !ok {
		var t T
		return t
	}
	return val
}

// GetRequestId retrieves the request ID from the context.
// Returns an empty string if no request ID is set.
//
// The request ID is typically a unique identifier assigned to each incoming
// HTTP request to track it through the system, useful for logging, tracing,
// and debugging.
//
// Example:
//
//	func LogHandler(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        requestID := ctxutil.GetRequestId(r.Context())
//	        log.Printf("[%s] Request received: %s", requestID, r.URL.Path)
//	        next.ServeHTTP(w, r)
//	    })
//	}
func GetRequestId(ctx context.Context) string {
	return getValue[string](ctx, request_id)
}

// SetRequestId adds or updates the request ID in the context.
// It returns a new context with the request ID value set.
//
// This function should typically be called early in the request handling
// lifecycle, such as in a middleware that assigns request IDs.
//
// Example:
//
//	func RequestIDMiddleware(next http.Handler) http.Handler {
//	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//	        // Generate or extract request ID
//	        requestID := uid.New(uid.RequestPrefix)
//
//	        // Add it to the context
//	        ctx := ctxutil.SetRequestId(r.Context(), requestID)
//
//	        // Set header for client tracking
//	        w.Header().Set("X-Request-ID", requestID)
//
//	        // Continue with the updated context
//	        next.ServeHTTP(w, r.WithContext(ctx))
//	    })
//	}
func SetRequestId(ctx context.Context, requestId string) context.Context {
	return context.WithValue(ctx, request_id, requestId)
}
