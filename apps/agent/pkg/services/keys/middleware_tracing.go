package keys

import (
	"context"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type tracingMiddleware struct {
	tracer tracing.Tracer
	next   KeyService
}

func WithTracing(tracer tracing.Tracer) Middleware {
	return func(svc KeyService) KeyService {
		return &tracingMiddleware{
			tracer: tracer,
			next:   svc,
		}
	}
}

func (mw *tracingMiddleware) CreateKey(ctx context.Context, req *authenticationv1.CreateKeyRequest) (*authenticationv1.CreateKeyResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "CreateKey"))
	defer span.End()

	res, err := mw.next.CreateKey(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) SoftDeleteKey(ctx context.Context, req *authenticationv1.SoftDeleteKeyRequest) (*authenticationv1.SoftDeleteKeyResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "SoftDeleteKey"))
	defer span.End()

	res, err := mw.next.SoftDeleteKey(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) VerifyKey(ctx context.Context, req *authenticationv1.VerifyKeyRequest) (*authenticationv1.VerifyKeyResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "VerifyKey"))
	defer span.End()

	res, err := mw.next.VerifyKey(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) GetKey(ctx context.Context, req *authenticationv1.GetKeyRequest) (*authenticationv1.GetKeyResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "GetKey"))
	defer span.End()

	res, err := mw.next.GetKey(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
