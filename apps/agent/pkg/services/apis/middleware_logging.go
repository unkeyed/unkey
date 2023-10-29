package apis

import (
	"context"
	"time"

	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
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

func (mw *loggingMiddleware) CreateApi(ctx context.Context, req *apisv1.CreateApiRequest) (*apisv1.CreateApiResponse, error) {
	start := time.Now()

	res, err := mw.next.CreateApi(ctx, req)
	mw.logger.Info().Str("method", "CreateApi").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.ApiService")
	return res, err
}
func (mw *loggingMiddleware) DeleteApi(ctx context.Context, req *apisv1.DeleteApiRequest) (*apisv1.DeleteApiResponse, error) {
	start := time.Now()

	res, err := mw.next.DeleteApi(ctx, req)
	mw.logger.Info().Str("method", "DeleteApi").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.ApiService")
	return res, err
}

func (mw *loggingMiddleware) FindApi(ctx context.Context, req *apisv1.FindApiRequest) (*apisv1.FindApiResponse, error) {
	start := time.Now()

	res, err := mw.next.FindApi(ctx, req)
	mw.logger.Info().Str("method", "FindApi").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.ApiService")
	return res, err
}
