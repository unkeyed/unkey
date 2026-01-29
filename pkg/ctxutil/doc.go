// Package ctxutil provides utilities for working with context.Context,
// specifically focused on storing and retrieving request-scoped values.
//
// This package implements a type-safe approach to context values by using
// strongly typed keys and accessor functions. This avoids the common pitfalls
// of using string literals or other non-type-safe methods for context keys.
//
// Key features:
// - Type-safe value retrieval
// - Strongly typed context keys
// - Zero-value defaults when values aren't present
//
// Common usage:
//
//	// Adding a request ID to context
//	ctx = ctxutil.SetRequestID(ctx, "req_123abc")
//
//	// Later, retrieving the request ID
//	requestID := ctxutil.GetRequestID(ctx)
//	fmt.Printf("Processing request: %s", requestID)
//
// This package should be used for passing request-scoped data that would be
// impractical to pass through function parameters, especially across API
// boundaries where modifying function signatures would be disruptive.
package ctxutil
