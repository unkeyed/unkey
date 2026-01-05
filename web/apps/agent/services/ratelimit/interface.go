package ratelimit

import (
	"context"

	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
)

type Service interface {
	Ratelimit(context.Context, *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error)
	MultiRatelimit(context.Context, *ratelimitv1.RatelimitMultiRequest) (*ratelimitv1.RatelimitMultiResponse, error)
	PushPull(context.Context, *ratelimitv1.PushPullRequest) (*ratelimitv1.PushPullResponse, error)
	CommitLease(context.Context, *ratelimitv1.CommitLeaseRequest) (*ratelimitv1.CommitLeaseResponse, error)
	Mitigate(context.Context, *ratelimitv1.MitigateRequest) (*ratelimitv1.MitigateResponse, error)
}

type Middleware func(Service) Service
