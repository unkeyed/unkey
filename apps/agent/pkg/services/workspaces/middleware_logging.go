package workspaces

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"go.uber.org/zap"
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
	mw.logger.Info("mw.CreateWorkspace", zap.String("method", "CreateWorkspace"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return res, err
}
func (mw *loggingMiddleware) ChangePlan(ctx context.Context, req ChangePlanRequest) (ChangePlanResponse, error) {
	start := time.Now()

	res, err := mw.next.ChangePlan(ctx, req)
	mw.logger.Info("mw.workspaceService", zap.String("method", "ChangePlan"), zap.Error(err), zap.Int64("latency", time.Since(start).Milliseconds()))
	return res, err
}
