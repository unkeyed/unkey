package ratelimit

import (
	"context"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func WithTracing(svc Service) Service {
	return &tracingMiddleware{next: svc}
}

type tracingMiddleware struct {
	next Service
}

func (mw *tracingMiddleware) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (res *ratelimitv1.RatelimitResponse, err error) {

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("svc.ratelimit", "Ratelimit"))
	defer span.End()
	span.SetAttributes(attribute.String("identifier", req.Identifier), attribute.String("name", req.Name))

	res, err = mw.next.Ratelimit(ctx, req)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return res, err
}

func (mw *tracingMiddleware) MultiRatelimit(ctx context.Context, req *ratelimitv1.RatelimitMultiRequest) (res *ratelimitv1.RatelimitMultiResponse, err error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("svc.ratelimit", "MultiRatelimit"))
	defer span.End()

	res, err = mw.next.MultiRatelimit(ctx, req)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return res, err
}

func (mw *tracingMiddleware) PushPull(ctx context.Context, req *ratelimitv1.PushPullRequest) (res *ratelimitv1.PushPullResponse, err error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("svc.ratelimit", "PushPull"))
	defer span.End()

	res, err = mw.next.PushPull(ctx, req)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return res, err
}

func (mw *tracingMiddleware) CommitLease(ctx context.Context, req *ratelimitv1.CommitLeaseRequest) (res *ratelimitv1.CommitLeaseResponse, err error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("svc.ratelimit", "CommitLease"))
	defer span.End()

	res, err = mw.next.CommitLease(ctx, req)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return res, err
}

func (mw *tracingMiddleware) Mitigate(ctx context.Context, req *ratelimitv1.MitigateRequest) (res *ratelimitv1.MitigateResponse, err error) {
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("svc.ratelimit", "Mitigate"))
	defer span.End()

	res, err = mw.next.Mitigate(ctx, req)
	if err != nil {
		tracing.RecordError(span, err)
	}
	return res, err
}
