package ratelimit

import (
	"context"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
)

type Service interface {
	Ratelimit(context.Context, *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error)
}

type Middleware func(Service) Service
