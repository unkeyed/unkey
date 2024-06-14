package connect

import (
	"context"

	"github.com/bufbuild/connect-go"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type Middleware func(ratelimitv1connect.RatelimitServiceHandler) ratelimitv1connect.RatelimitServiceHandler

type tracingMiddleware struct {
	tracer tracing.Tracer
	next   ratelimitv1connect.RatelimitServiceHandler
}

func WithTracing(tracer tracing.Tracer) Middleware {
	return func(svc ratelimitv1connect.RatelimitServiceHandler) ratelimitv1connect.RatelimitServiceHandler {
		return &tracingMiddleware{
			tracer: tracer,
			next:   svc,
		}
	}
}

func (mw *tracingMiddleware) Liveness(
	ctx context.Context,
	req *connect.Request[ratelimitv1.LivenessRequest],
) (*connect.Response[ratelimitv1.LivenessResponse], error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("ratelimit", "Liveness"))
	defer span.End()

	res, err := mw.next.Liveness(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
func (mw *tracingMiddleware) Ratelimit(
	ctx context.Context,
	req *connect.Request[ratelimitv1.RatelimitRequest],
) (*connect.Response[ratelimitv1.RatelimitResponse], error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("ratelimit", "Ratelimit"))
	defer span.End()
	span.SetAttributes(attribute.String("identifier", req.Msg.Identifier))
	span.SetAttributes(attribute.Int64("limit", req.Msg.Limit))
	span.SetAttributes(attribute.Int64("duration", req.Msg.Duration))
	span.SetAttributes(attribute.Int64("cost", req.Msg.Cost))

	res, err := mw.next.Ratelimit(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
