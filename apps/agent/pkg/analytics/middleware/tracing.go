package middleware

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type tracingMiddleware struct {
	next   analytics.Analytics
	tracer tracing.Tracer
}

func WithTracing(next analytics.Analytics, tracer tracing.Tracer) analytics.Analytics {
	return &tracingMiddleware{next: next, tracer: tracer}
}

func (mw *tracingMiddleware) PublishKeyVerificationEvent(ctx context.Context, event analytics.KeyVerificationEvent) {
	ctx, span := mw.tracer.Start(ctx, "analytics.PublishKeyVerificationEvent", trace.WithAttributes(attribute.String("keyId", event.KeyId)))
	defer span.End()
	mw.next.PublishKeyVerificationEvent(ctx, event)

}
func (mw *tracingMiddleware) GetKeyStats(ctx context.Context, keyId string) (analytics.KeyStats, error) {
	ctx, span := mw.tracer.Start(ctx, "analytics.GetKeyStats", trace.WithAttributes(attribute.String("keyId", keyId)))
	defer span.End()

	stats, err := mw.next.GetKeyStats(ctx, keyId)
	if err != nil {
		span.RecordError(err)
	}
	return stats, err

}
