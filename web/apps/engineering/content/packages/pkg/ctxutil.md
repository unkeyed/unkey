---
title: ctxutil
description: "provides utilities for working with context.Context,"
---

Package ctxutil provides utilities for working with context.Context, specifically focused on storing and retrieving request-scoped values.

This package implements a type-safe approach to context values by using strongly typed keys and accessor functions. This avoids the common pitfalls of using string literals or other non-type-safe methods for context keys.

Key features: - Type-safe value retrieval - Strongly typed context keys - Zero-value defaults when values aren't present

Common usage:

	// Adding a request ID to context
	ctx = ctxutil.SetRequestID(ctx, "req_123abc")

	// Later, retrieving the request ID
	requestID := ctxutil.GetRequestID(ctx)
	fmt.Printf("Processing request: %s", requestID)

This package should be used for passing request-scoped data that would be impractical to pass through function parameters, especially across API boundaries where modifying function signatures would be disruptive.

## Functions

### func GetRequestID

```go
func GetRequestID(ctx context.Context) string
```

GetRequestID retrieves the request ID from the context. Returns an empty string if no request ID is set.

The request ID is typically a unique identifier assigned to each incoming HTTP request to track it through the system, useful for logging, tracing, and debugging.

Example:

	func LogHandler(next http.Handler) http.Handler {
	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	        requestID := ctxutil.GetRequestID(r.Context())
	        log.Printf("[%s] Request received: %s", requestID, r.URL.Path)
	        next.ServeHTTP(w, r)
	    })
	}

### func SetRequestID

```go
func SetRequestID(ctx context.Context, requestID string) context.Context
```

SetRequestID adds or updates the request ID in the context. It returns a new context with the request ID value set.

This function should typically be called early in the request handling lifecycle, such as in a middleware that assigns request IDs.

Example:

	func RequestIDMiddleware(next http.Handler) http.Handler {
	    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	        // Generate or extract request ID
	        requestID := uid.New(uid.RequestPrefix)

	        // Add it to the context
	        ctx := ctxutil.SetRequestID(r.Context(), requestID)

	        // Set header for client tracking
	        w.Header().Set("X-Request-ID", requestID)

	        // Continue with the updated context
	        next.ServeHTTP(w, r.WithContext(ctx))
	    })
	}


## Types

