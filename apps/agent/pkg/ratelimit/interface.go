package ratelimit

import (
	"context"
	"time"
)

type Ratelimiter interface {
	Take(ctx context.Context, req RatelimitRequest) RatelimitResponse
	SetCurrent(ctx context.Context, req SetCurrentRequest) error
	CommitLease(ctx context.Context, req CommitLeaseRequest) error
}

type Lease struct {
	Cost      int64
	ExpiresAt time.Time
}

type RatelimitRequest struct {
	Name       string
	Identifier string
	Limit      int64
	Cost       int64
	Duration   time.Duration
	Lease      *Lease
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
	Limit      int64

	Duration time.Duration
	Current  int64
}

type CommitLeaseRequest struct {
	Identifier string
	LeaseId    string
	Tokens     int64
}
