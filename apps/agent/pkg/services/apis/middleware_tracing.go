package apis

import (
	"context"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
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

func (mw *tracingMiddleware) CreateApi(ctx context.Context, req *apisv1.CreateApiRequest) (*apisv1.CreateApiResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "CreateApi"))
	defer span.End()

	res, err := mw.next.CreateApi(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) DeleteApi(ctx context.Context, req *apisv1.DeleteApiRequest) (*apisv1.DeleteApiResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "DeleteApi"))
	defer span.End()

	res, err := mw.next.DeleteApi(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) FindApi(ctx context.Context, req *apisv1.FindApiRequest) (*apisv1.FindApiResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "FindApi"))
	defer span.End()

	res, err := mw.next.FindApi(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
