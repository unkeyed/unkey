package usagelimiter

import (
	"context"
)

// Service defines the interface for usage limiting operations. It enforces
// credit-based rate limits on API keys by tracking and decrementing available
// credits. Implementations may use direct database queries or distributed
// Redis counters for higher performance.
type Service interface {
	// Limit checks whether the given key has sufficient credits to cover the
	// requested cost and, if so, decrements the credits atomically. Returns a
	// [UsageResponse] indicating whether the request is valid and how many
	// credits remain.
	Limit(ctx context.Context, req UsageRequest) (UsageResponse, error)

	// Invalidate removes the cached limit for the given keyID, forcing the
	// next [Limit] call to reload from the database.
	Invalidate(ctx context.Context, keyID string) error

	// Close gracefully shuts down the usage limiter service, draining any
	// pending replay operations before returning.
	Close() error
}

// UsageRequest represents a request to check and decrement usage credits
// for an API key. The Cost field specifies how many credits this operation
// should consume.
type UsageRequest struct {
	KeyID string
	Cost  int32
}

// UsageResponse represents the result of a usage limit check. Valid indicates
// whether the key has sufficient credits for the requested operation.
// Remaining reports the number of credits left after the operation, or -1
// if the key has no limit configured.
type UsageResponse struct {
	Valid     bool
	Remaining int32 // Remaining usage for the keyID -1 indicates no limit
}
