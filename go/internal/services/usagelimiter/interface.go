package usagelimiter

import (
	"context"
)

type Service interface {
	// If the given Identifier has exceeded its usage limit, an error is returned.
	Limit(ctx context.Context, req UsageRequest) (UsageResponse, error)

	// Invalidate removes the cached limit for the given Identifier.
	// The Identifier parameter accepts either a KeyID (for legacy key-based credits)
	// or a CreditID (for new credits system). The implementation attempts to invalidate
	// both formats since the caller may not know which system is in use.
	// Examples: "key_123abc" or "credit_456def"
	Invalidate(ctx context.Context, Identifier string) error

	// Close gracefully shuts down the usage limiter service.
	Close() error
}

type UsageRequest struct {
	// For legacy key-based credits stored in keys.remaining_requests
	KeyID string
	// For new credits system stored in credits table.
	// When present, this takes precedence over KeyID.
	CreditID string
	Cost     int32
}

type UsageResponse struct {
	Valid     bool
	Remaining int32 // Remaining usage for the Identifier -1 indicates no limit
}
