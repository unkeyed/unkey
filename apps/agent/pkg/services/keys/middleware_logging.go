package keys

import (
	"context"
	"time"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
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

func (mw *loggingMiddleware) CreateKey(ctx context.Context, req *authenticationv1.CreateKeyRequest) (*authenticationv1.CreateKeyResponse, error) {
	mw.logger.Info().Str("method", "CreateKey").Msg("called")
	start := time.Now()

	res, err := mw.next.CreateKey(ctx, req)
	mw.logger.Info().Str("method", "CreateKey").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.KeyService")
	return res, err
}

func (mw *loggingMiddleware) SoftDeleteKey(ctx context.Context, req *authenticationv1.SoftDeleteKeyRequest) (*authenticationv1.SoftDeleteKeyResponse, error) {
	mw.logger.Info().Str("method", "SoftDeleteKey").Msg("called")
	start := time.Now()

	res, err := mw.next.SoftDeleteKey(ctx, req)
	mw.logger.Info().Str("method", "SoftDeleteKey").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.KeyService")
	return res, err
}

func (mw *loggingMiddleware) VerifyKey(ctx context.Context, req *authenticationv1.VerifyKeyRequest) (*authenticationv1.VerifyKeyResponse, error) {
	mw.logger.Info().Str("method", "VerifyKey").Msg("called")
	start := time.Now()

	res, err := mw.next.VerifyKey(ctx, req)
	mw.logger.Info().Str("method", "VerifyKey").Err(err).Int64("latency", time.Since(start).Milliseconds()).Msg("mw.KeyService")
	return res, err
}
