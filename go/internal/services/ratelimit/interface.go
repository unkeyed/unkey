package ratelimit

import (
	"context"
	"time"
)

type Service interface {
	Ratelimit(context.Context, RatelimitRequest) (RatelimitResponse, error)
}

// RatelimitRequest represents a request to check or consume rate limit tokens.
// This is typically the first point of contact when a client wants to verify
// if they are allowed to perform an action under the rate limit constraints.
type RatelimitRequest struct {
	// Unique identifier for the rate limit subject.
	// This could be:
	// - A user ID
	// - An API key
	// - An IP address
	// - Any other unique identifier that needs rate limiting
	Identifier string

	// Maximum number of tokens allowed within the duration.
	// Once this limit is reached, subsequent requests will be denied
	// until there is more capacity.
	Limit int64

	// Duration of the rate limit window in milliseconds.
	// After this duration, a new window begins.
	// Common values might be:
	// - 1000 (1 second)
	// - 60000 (1 minute)
	// - 3600000 (1 hour)
	Duration time.Duration

	// Number of tokens to consume in this request.
	// Defaults to 1 if not specified.
	// Higher values can be used for operations that should count more heavily
	// against the rate limit (e.g., batch operations).
	Cost int64
}

// RatelimitResponse contains the result of a rate limit check.
// This response includes all necessary information for clients to understand
// their current rate limit status and when they can retry if limited.
type RatelimitResponse struct {
	// Total limit configured for this window.
	// This matches the limit specified in the request and is included
	// for convenience in client implementations.
	Limit int64

	// Number of tokens remaining in the current window.
	// Clients can use this to implement progressive backoff or
	// warn users when they're close to their limit.
	Remaining int64

	// Unix timestamp (in milliseconds) when the current window expires.
	// Clients can use this to:
	// - Display time until reset to users
	// - Implement automatic retry after window reset
	// - Schedule future requests optimally
	Reset int64

	// Whether the rate limit check was successful.
	// true = request is allowed
	// false = request is denied due to rate limit exceeded
	Success bool

	// Current token count in this window.
	// This represents how many tokens have been consumed so far,
	// useful for monitoring and debugging purposes.
	Current int64
}
type Middleware func(Service) Service
