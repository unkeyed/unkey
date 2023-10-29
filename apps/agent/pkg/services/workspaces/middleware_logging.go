package workspaces

import (
	"context"
	"time"

	workspacesv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/workspaces/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type loggingMiddleware struct {
	logger logging.Logger
	next   WorkspaceService
}

func WithLogging(logger logging.Logger) Middleware {
	return func(svc WorkspaceService) WorkspaceService {
		return &loggingMiddleware{
			logger: logger,
			next:   svc,
		}
	}
}

func (mw *loggingMiddleware) CreateWorkspace(ctx context.Context, req *workspacesv1.CreateWorkspaceRequest) (*workspacesv1.CreateWorkspaceResponse, error) {
	start := time.Now()

	res, err := mw.next.CreateWorkspace(ctx, req)
	mw.logger.Info().Str("method", "CreateWorkspace").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}
func (mw *loggingMiddleware) ChangePlan(ctx context.Context, req *workspacesv1.ChangePlanRequest) (*workspacesv1.ChangePlanResponse, error) {
	start := time.Now()

	res, err := mw.next.ChangePlan(ctx, req)
	mw.logger.Info().Str("method", "ChangePlan").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}

func (mw *loggingMiddleware) DeleteWorkspace(ctx context.Context, req *workspacesv1.DeleteWorkspaceRequest) (*workspacesv1.DeleteWorkspaceResponse, error) {
	start := time.Now()

	res, err := mw.next.DeleteWorkspace(ctx, req)
	mw.logger.Info().Str("method", "DeleteWorkspace").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}

func (mw *loggingMiddleware) FindWorkspace(ctx context.Context, req *workspacesv1.FindWorkspaceRequest) (*workspacesv1.FindWorkspaceResponse, error) {
	start := time.Now()

	res, err := mw.next.FindWorkspace(ctx, req)
	mw.logger.Info().Str("method", "FindWorkspace").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}

func (mw *loggingMiddleware) RenameWorkspace(ctx context.Context, req *workspacesv1.RenameWorkspaceRequest) (*workspacesv1.RenameWorkspaceResponse, error) {
	start := time.Now()

	res, err := mw.next.RenameWorkspace(ctx, req)
	mw.logger.Info().Str("method", "RenameWorkspace").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}
