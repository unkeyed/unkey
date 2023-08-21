package workspaces

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type tracingMiddleware struct {
	tracer tracing.Tracer
	next   WorkspaceService
}

func WithTracing(tracer tracing.Tracer) Middleware {
	return func(svc WorkspaceService) WorkspaceService {
		return &tracingMiddleware{
			tracer: tracer,
			next:   svc,
		}
	}
}

func (mw *tracingMiddleware) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (CreateWorkspaceResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "CreateWorkspace"))
	defer span.End()

	res, err := mw.next.CreateWorkspace(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) ChangePlan(ctx context.Context, req ChangePlanRequest) (ChangePlanResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "ChangePlan"))
	defer span.End()

	res, err := mw.next.ChangePlan(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
