package workspaces

import (
	"context"
	"time"

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

func (mw *loggingMiddleware) CreateWorkspace(ctx context.Context, req CreateWorkspaceRequest) (CreateWorkspaceResponse, error) {
	start := time.Now()

	res, err := mw.next.CreateWorkspace(ctx, req)
	mw.logger.Info().Str("method", "CreateWorkspace").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}
func (mw *loggingMiddleware) ChangePlan(ctx context.Context, req ChangePlanRequest) (ChangePlanResponse, error) {
	start := time.Now()

	res, err := mw.next.ChangePlan(ctx, req)
	mw.logger.Info().Str("method", "ChangePlan").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.workspaceService")
	return res, err
}
