package ratelimit

import (
	"context"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
)

type Service interface {
	Ratelimit(context.Context, *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error)
	MultiRatelimit(context.Context, *ratelimitv1.RatelimitMultiRequest) (*ratelimitv1.RatelimitMultiResponse, error)
	PushPull(context.Context, *ratelimitv1.PushPullRequest) (*ratelimitv1.PushPullResponse, error)
}

type Middleware func(Service) Service
