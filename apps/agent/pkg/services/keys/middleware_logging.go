package keys

import (
	"context"
	"time"

	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type loggingMiddleware struct {
	logger logging.Logger
	next   KeyService
}

func WithLogging(logger logging.Logger) Middleware {
	return func(svc KeyService) KeyService {
		return &loggingMiddleware{
			logger: logger,
			next:   svc,
		}
	}
}

func (mw *loggingMiddleware) CreateKey(ctx context.Context, req *keysv1.CreateKeyRequest) (*keysv1.CreateKeyResponse, error) {
	mw.logger.Info().Str("method", "CreateKey").Msg("called")
	start := time.Now()

	res, err := mw.next.CreateKey(ctx, req)
	mw.logger.Info().Str("method", "CreateKey").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.KeyService")
	return res, err
}
