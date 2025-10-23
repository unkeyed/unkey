package usagelimiter

import (
	"context"
)

type Service interface {
	// If the given Identifier has exceeded its usage limit, an error is returned.
	Limit(ctx context.Context, req UsageRequest) (UsageResponse, error)

	// Invalidate removes the cached limit for the given Identifier.
	Invalidate(ctx context.Context, Identifier string) error

	// Close gracefully shuts down the usage limiter service.
	Close() error
}

type UsageRequest struct {
	CreditID string
	Cost     int32
}

type UsageResponse struct {
	Valid     bool
	Remaining int32 // Remaining usage for the Identifier -1 indicates no limit
}
