// Package ratelimit provides distributed rate limiting functionality using a sliding window algorithm.
package ratelimit

import (
	"context"
	"time"
)

// Service defines the core rate limiting functionality. It provides thread-safe
// operations for checking and consuming rate limit tokens across a distributed system.
//
// The service maintains rate limit state using a sliding window algorithm and
// propagates updates across cluster nodes to maintain eventual consistency.
//
// Concurrency: All methods are safe for concurrent use.
type Service interface {
	// Ratelimit checks if a request should be allowed under the current rate limit constraints
	// and consumes tokens if successful. It uses a sliding window algorithm that considers
	// both the current and previous time windows to provide accurate rate limiting.
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//   - req: The rate limit request parameters
	//
	// Returns:
	//   - RatelimitResponse: Contains the result and current rate limit state
	//   - error: Any validation or system errors that occurred
	//
	// Errors:
	//   - Returns validation errors for invalid request parameters
	//   - May return errors related to cluster communication
	//
	// Performance: O(1) time complexity for local decisions
	//
	// Example Usage:
	//   response, err := svc.Ratelimit(ctx, RatelimitRequest{
	//     Identifier: "user-123",
	//     Limit:      100,
	//     Duration:   time.Minute,
	//     Cost:       1,
	//   })
	Ratelimit(context.Context, RatelimitRequest) (RatelimitResponse, error)

	RatelimitMany(context.Context, []RatelimitRequest) ([]RatelimitResponse, error)
}

// RatelimitRequest represents a request to check or consume rate limit tokens.
// This is typically the first point of contact when a client wants to verify
// if they are allowed to perform an action under the rate limit constraints.
//
// The request combines an identifier with limit parameters to uniquely identify
// and control a rate limit bucket. Multiple requests with the same parameters
// will operate on the same underlying rate limit state.
//
// Thread Safety: This type is immutable and safe for concurrent use.
type RatelimitRequest struct {
	// WorkspaceID scopes every other field of this request. Two workspaces
	// using the same Namespace + Identifier are kept fully isolated; this
	// matters both for correctness (no cross-tenant counter pollution) and
	// for cross-region propagation, which is keyed on WorkspaceID.
	//
	// Must be non-empty.
	WorkspaceID string

	// Namespace identifies the rate limit topic within a workspace. It is an
	// opaque string scoped to WorkspaceID, not necessarily a row ID in
	// ratelimit_namespaces. The standalone ratelimit API uses the namespace
	// row ID; key-bound limits use the user-defined ratelimit config name
	// (e.g. "tokens"); workspace-level ratelimits use a constant. Two
	// namespaces with the same string in different workspaces are isolated.
	Namespace string

	// Identifier uniquely identifies the rate limit subject.
	// This could be:
	//   - A user ID
	//   - An API key ID
	//   - An IP address
	//   - Any other unique identifier that needs rate limiting
	//
	// Must be non-empty. The same identifier with different Duration
	// values will be treated as separate rate limits.
	Identifier string

	// Limit specifies the maximum number of tokens allowed within the Duration.
	// Once this limit is reached, subsequent requests will be denied until the
	// window rolls over.
	//
	// Must be > 0. Common values:
	//   -  1_000 for per-second limits
	//   - 60_000 for per-minute limits
	Limit int64

	// Duration specifies the time window for the rate limit.
	// After this duration, a new window begins and the token count resets.
	//
	// Must be >= 1 second. Common values:
	//   - time.Second
	//   - time.Minute
	//   - time.Hour
	Duration time.Duration

	// Cost specifies the number of tokens to consume in this request.
	// Higher values can be used for operations that should count more
	// heavily against the rate limit (e.g., batch operations).
	//
	// Must be >= 0. Defaults to 1 if not specified.
	// The request will be denied if Cost > Limit.
	Cost int64

	// Time of the request
	// If not specified or if zero, the ratelimiter will use its own clock.
	Time time.Time
}

// RatelimitResponse contains the result of a rate limit check and the current state
// of the rate limit window. This response provides all necessary information for clients
// to understand their current rate limit status and implement appropriate behavior.
//
// Thread Safety: This type is immutable and safe for concurrent use.
type RatelimitResponse struct {
	// Limit is the total number of tokens allowed in the current window.
	// This matches the limit specified in the request and is included
	// for convenience in client implementations.
	Limit int64

	// Remaining is the number of tokens still available in the current window.
	// Clients can use this to:
	//   - Implement progressive backoff
	//   - Warn users when approaching limits
	//   - Make decisions about request priorities
	//
	// Will be 0 when the rate limit is exceeded.
	Remaining int64

	// Reset is the Unix timestamp (in milliseconds) when the current window expires.
	// Clients can use this to:
	//   - Display time until reset to users
	//   - Implement automatic retry after window reset
	//   - Schedule future requests optimally
	//   - Calculate backoff periods
	Reset time.Time

	// Success indicates whether the rate limit check passed.
	//   true  = request is allowed
	//   false = request is denied (rate limit exceeded)
	//
	// When false, clients should use Reset to determine when to retry.
	Success bool

	// Current represents how many tokens have been consumed in this window.
	// This is useful for:
	//   - Monitoring and debugging
	//   - Understanding usage patterns
	//   - Implementing custom backoff strategies
	Current int64
}

// Middleware defines a function type that wraps a Service with additional functionality.
// It can be used to add logging, metrics, validation, or other cross-cutting concerns.
//
// Example Usage:
//
//	func LoggingMiddleware(logger Logger) Middleware {
//	    return func(next Service) Service {
//	        return &loggingService{next: next}
//	    }
//	}
type Middleware func(Service) Service
