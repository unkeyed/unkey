package workspaces

import (
	"context"
	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"

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

func (mw *tracingMiddleware) CreateWorkspace(ctx context.Context, req *workspacesv1.CreateWorkspaceRequest) (*workspacesv1.CreateWorkspaceResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "CreateWorkspace"))
	defer span.End()

	res, err := mw.next.CreateWorkspace(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) ChangePlan(ctx context.Context, req *workspacesv1.ChangePlanRequest) (*workspacesv1.ChangePlanResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "ChangePlan"))
	defer span.End()

	res, err := mw.next.ChangePlan(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) RenameWorkspace(ctx context.Context, req *workspacesv1.RenameWorkspaceRequest) (*workspacesv1.RenameWorkspaceResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "RenameWorkspace"))
	defer span.End()

	res, err := mw.next.RenameWorkspace(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) DeleteWorkspace(ctx context.Context, req *workspacesv1.DeleteWorkspaceRequest) (*workspacesv1.DeleteWorkspaceResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "DeleteWorkspace"))
	defer span.End()

	res, err := mw.next.DeleteWorkspace(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}

func (mw *tracingMiddleware) FindWorkspace(ctx context.Context, req *workspacesv1.FindWorkspaceRequest) (*workspacesv1.FindWorkspaceResponse, error) {
	ctx, span := mw.tracer.Start(ctx, tracing.NewSpanName("workspaces", "FindWorkspace"))
	defer span.End()

	res, err := mw.next.FindWorkspace(ctx, req)
	if err != nil {
		span.RecordError(err)
	}
	return res, err
}
