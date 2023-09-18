package apis

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type tracingMiddleware struct {
	tracer tracing.Tracer
	next   ApiService
}

func WithTracing(tracer tracing.Tracer) Middleware {
	return func(svc ApiService) ApiService {
		return &tracingMiddleware{
			tracer: tracer,
			next:   svc,
		}
	}
}

func (mw *tracingMiddleware) CreateApi(ctx context.Context, req CreateApiRequest) (CreateApiResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "CreateApi"))
	defer span.End()

	res, err := mw.next.CreateApi(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) RemoveApi(ctx context.Context, req RemoveApiRequest) (RemoveApiResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "RemoveApi"))
	defer span.End()

	res, err := mw.next.RemoveApi(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
