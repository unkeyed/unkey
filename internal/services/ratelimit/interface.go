// Package ratelimit provides distributed rate limiting with sliding-window counters.
package ratelimit

import (
	"context"
	"time"
)

// Service checks and consumes rate-limit tokens.
//
// Implementations must be safe for concurrent use. The concrete service in
// this package enforces a sliding window locally, converges nodes within a
// region through Redis, and imports foreign-region counts through MySQL.
type Service interface {
	// Ratelimit checks one limit and consumes req.Cost tokens when the request
	// fits in the sliding window. A denied request returns a nil error with
	// RatelimitResponse.Success set to false; validation failures return an
	// empty response and a non-nil error.
	Ratelimit(context.Context, RatelimitRequest) (RatelimitResponse, error)

	// RatelimitMany checks a batch as one all-or-nothing operation. Each response
	// reports whether that specific request fit its own limit, but counter side
	// effects are committed only when every request passes. If any request fails,
	// all optimistic increments are rolled back before the method returns.
	RatelimitMany(context.Context, []RatelimitRequest) ([]RatelimitResponse, error)
}

// RatelimitRequest describes one sliding-window rate-limit check.
//
// WorkspaceID, Namespace, Identifier, Duration, and Time select the window cell.
// Limit and Cost determine whether that cell can accept more work. The value is
// immutable by convention after it is passed to [Service.Ratelimit] or
// [Service.RatelimitMany].
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

	// Identifier uniquely identifies the rate-limit subject, such as a user ID,
	// API key ID, or IP address.
	//
	// Must be non-empty. The same identifier with different Duration
	// values will be treated as separate rate limits.
	Identifier string

	// Limit specifies the maximum number of tokens allowed within Duration.
	// Once this limit is reached, subsequent requests will be denied until the
	// window rolls over.
	//
	// Must be greater than 0.
	Limit int64

	// Duration specifies the time window for the rate limit.
	// After this duration, a new window begins and the token count resets.
	//
	// Must be at least 1 second.
	Duration time.Duration

	// Cost specifies the number of tokens to consume in this request.
	// Higher values can be used for operations that should count more
	// heavily against the rate limit, such as batch operations.
	//
	// Must be non-negative. A zero cost performs a check without consuming
	// tokens. The service does not default zero to one.
	Cost int64

	// Time is the request timestamp used to choose the sliding-window sequence.
	// If zero, the service uses its own clock.
	Time time.Time
}

// RatelimitResponse contains the result of a rate-limit check.
//
// The response is immutable by convention. Success false means the request was
// denied by the limit, not that the service failed; system and validation
// failures are returned as errors instead.
type RatelimitResponse struct {
	// Limit is the total number of tokens allowed in the current window.
	// This matches the limit specified in the request and is included
	// for convenience in client implementations.
	Limit int64

	// Remaining is the number of tokens still available in the current window.
	// It is 0 when the rate limit is exceeded.
	Remaining int64

	// Reset is when the current fixed window expires. Sliding-window math still
	// carries a weighted portion of the previous window after this timestamp.
	Reset time.Time

	// Success indicates whether the rate limit check passed.
	// When false, callers can use Reset to decide when to retry.
	Success bool

	// Current is the effective sliding-window count after applying the request
	// cost. It includes the weighted previous window and imported cross-region
	// counts when those values are present.
	Current int64
}

// Middleware wraps a [Service] with cross-cutting behavior such as logging,
// metrics, or validation.
type Middleware func(Service) Service
