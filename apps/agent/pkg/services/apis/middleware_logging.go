package apis

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type loggingMiddleware struct {
	logger logging.Logger
	next   ApiService
}

func WithLogging(logger logging.Logger) Middleware {
	return func(svc ApiService) ApiService {
		return &loggingMiddleware{
			logger: logger,
			next:   svc,
		}
	}
}

func (mw *loggingMiddleware) CreateApi(ctx context.Context, req CreateApiRequest) (CreateApiResponse, error) {
	start := time.Now()

	res, err := mw.next.CreateApi(ctx, req)
	mw.logger.Info().Str("method", "CreateApi").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.ApiService")
	return res, err
}
func (mw *loggingMiddleware) RemoveApi(ctx context.Context, req RemoveApiRequest) (RemoveApiResponse, error) {
	start := time.Now()

	res, err := mw.next.RemoveApi(ctx, req)
	mw.logger.Info().Str("method", "RemoveApi").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.ApiService")
	return res, err
}
