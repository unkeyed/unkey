package ratelimit

import "context"

type Ratelimiter interface {
	Take(ctx context.Context, req RatelimitRequest) RatelimitResponse
	SetCurrent(ctx context.Context, req SetCurrentRequest) error
}

type RatelimitRequest struct {
	Identifier     string
	Max            int64
	Cost           int64
	RefillRate     int64
	RefillInterval int64
}

type RatelimitResponse struct {
	Pass      bool
	Limit     int64
	Remaining int64
	Reset     int64
	Current   int64
}

type SetCurrentRequest struct {
	Identifier string
	Max        int64

	RefillInterval int64
	Current        int64
}
