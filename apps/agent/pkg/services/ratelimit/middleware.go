package ratelimit

import (
	"context"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func WithTracing(tracer tracing.Tracer) Middleware {

	return func(svc Service) Service {
		return &tracingMiddleware{next: svc, tracer: tracer}
	}
}

type tracingMiddleware struct {
	next   Service
	tracer tracing.Tracer
}

func (mw *tracingMiddleware) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (res *ratelimitv1.RatelimitResponse, err error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("ratelimit", "Ratelimit"))
	defer span.End()

	res, err = mw.next.Ratelimit(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
