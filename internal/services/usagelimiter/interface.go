package usagelimiter

import (
	"context"
)

type Service interface {
	// If the given keyID has exceeded its usage limit, an error is returned.
	Limit(ctx context.Context, req UsageRequest) (UsageResponse, error)

	// Invalidate removes the cached limit for the given keyID.
	Invalidate(ctx context.Context, keyID string) error

	// Close gracefully shuts down the usage limiter service.
	Close() error
}

type UsageRequest struct {
	KeyID string
	Cost  int32
}

type UsageResponse struct {
	Valid     bool
	Remaining int32 // Remaining usage for the keyID -1 indicates no limit
}
