package usagelimiter

import (
	"context"
)

type Service interface {
	// If the given keyId has exceeded its usage limit, an error is returned.
	Limit(ctx context.Context, req UsageRequest) (UsageResponse, error)
}

type UsageRequest struct {
	KeyId string
	Cost  int32
}

type UsageResponse struct {
	Valid     bool
	Remaining int32 // Remaining usage for the keyId -1 indicates no limit
}
