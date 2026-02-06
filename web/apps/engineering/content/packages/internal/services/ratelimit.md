---
title: ratelimit
description: "implements a distributed rate limiting system using a sliding window algorithm"
---

Package ratelimit implements a distributed rate limiting system using a sliding window algorithm with cluster-wide state synchronization. It provides precise rate limiting across multiple nodes while maintaining low latency through local decision making and asynchronous state propagation.

### Architecture

The rate limiter uses a sliding window algorithm with the following key components:

  - Buckets: Track rate limit state for each unique identifier+limit+duration combination
  - Windows: Time-based counters that slide to maintain accurate request counts
  - Origin Nodes: Designated by consistent hashing to be the source of truth for each identifier
  - State Propagation: Asynchronous updates to maintain eventual consistency across the cluster

### Rate Limit Algorithm

The sliding window implementation:

 1. Maintains separate counters for current and previous time windows
 2. Calculates effective request count by combining: - 100% of current window - Weighted portion of previous window based on elapsed time
 3. Uses consistent hashing to route requests to origin nodes
 4. Propagates state changes asynchronously to maintain cluster-wide consistency

### Usage

To create a new rate limiting service:

	svc, err := ratelimit.New(ratelimit.Config{
	    Cluster: cluster,
	    Clock:   clock,
	})

To check if a request is allowed:

	resp, err := svc.Ratelimit(ctx, RatelimitRequest{
	    Identifier: "user-123",
	    Limit:      100,
	    Duration:   time.Minute,
	    Cost:      1,
	})

### Thread Safety

The package is designed to be thread-safe and can handle concurrent requests across multiple goroutines and nodes. All state modifications are protected by appropriate mutex locks.

### Error Handling

The service handles various error conditions:

  - Invalid configurations (negative limits, zero durations)
  - Network partitions between nodes
  - Node failures and cluster changes
  - Race conditions in distributed state

See the RatelimitRequest and RatelimitResponse types for detailed documentation of the API contract and error conditions.

Package ratelimit provides distributed rate limiting functionality using a sliding window algorithm.

## Functions


## Types

### type Config

```go
type Config struct {

	// Clock for time-related operations. If nil, uses system clock.
	Clock clock.Clock

	// Counter is the distributed counter backend (typically Redis).
	// Required - rate limiting cannot function without a counter.
	Counter counter.Counter
}
```

Config holds configuration for creating a new rate limiting service.

### type Middleware

```go
type Middleware func(Service) Service
```

Middleware defines a function type that wraps a Service with additional functionality. It can be used to add logging, metrics, validation, or other cross-cutting concerns.

Example Usage:

	func LoggingMiddleware(logger Logger) Middleware {
	    return func(next Service) Service {
	        return &loggingService{next: next}
	    }
	}

### type RatelimitRequest

```go
type RatelimitRequest struct {
	// Name is an arbitrary string that identifies the rate limit topic.
	Name string

	// Identifier uniquely identifies the rate limit subject.
	// This could be:
	//   - A user ID
	//   - An API key ID
	//   - An IP address
	//   - Any other unique identifier that needs rate limiting
	//
	// Must be non-empty. The same identifier with different Limit/Duration
	// combinations will be treated as separate rate limits.
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
	// If not specified or zero, the ratelimiter will use its own clock.
	Time time.Time
}
```

RatelimitRequest represents a request to check or consume rate limit tokens. This is typically the first point of contact when a client wants to verify if they are allowed to perform an action under the rate limit constraints.

The request combines an identifier with limit parameters to uniquely identify and control a rate limit bucket. Multiple requests with the same parameters will operate on the same underlying rate limit state.

Thread Safety: This type is immutable and safe for concurrent use.

### type RatelimitResponse

```go
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
```

RatelimitResponse contains the result of a rate limit check and the current state of the rate limit window. This response provides all necessary information for clients to understand their current rate limit status and implement appropriate behavior.

Thread Safety: This type is immutable and safe for concurrent use.

### type Service

```go
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
```

Service defines the core rate limiting functionality. It provides thread-safe operations for checking and consuming rate limit tokens across a distributed system.

The service maintains rate limit state using a sliding window algorithm and propagates updates across cluster nodes to maintain eventual consistency.

Concurrency: All methods are safe for concurrent use.

