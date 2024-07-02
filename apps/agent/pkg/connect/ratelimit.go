package connect

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bufbuild/connect-go"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

type ratelimitServer struct {
	svc    ratelimit.Service
	logger logging.Logger
	ratelimitv1connect.UnimplementedRatelimitServiceHandler
}

func NewRatelimitServer(svc ratelimit.Service, logger logging.Logger) *ratelimitServer {

	return &ratelimitServer{
		svc:    svc,
		logger: logger,
	}

}

func (s *ratelimitServer) CreateHandler() (string, http.Handler) {
	return ratelimitv1connect.NewRatelimitServiceHandler(s)

}

func (s *ratelimitServer) Ratelimit(
	ctx context.Context,
	req *connect.Request[ratelimitv1.RatelimitRequest],
) (*connect.Response[ratelimitv1.RatelimitResponse], error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("connect.Ratelimit")
	}()

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "Ratelimit"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.Ratelimit(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to ratelimit: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *ratelimitServer) MultiRatelimit(
	ctx context.Context,
	req *connect.Request[ratelimitv1.RatelimitMultiRequest],
) (*connect.Response[ratelimitv1.RatelimitMultiResponse], error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("connect.MultiRatelimit")
	}()

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "MultiRatelimit"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.MultiRatelimit(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to ratelimit: %w", err)
	}
	return connect.NewResponse(res), nil

}

func (s *ratelimitServer) PushPull(
	ctx context.Context,
	req *connect.Request[ratelimitv1.PushPullRequest],
) (*connect.Response[ratelimitv1.PushPullResponse], error) {
	start := time.Now()
	defer func() {
		s.logger.Info().
			Int64("latency", time.Since(start).Milliseconds()).
			Msg("connect.PushPull")
	}()
	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "PushPull"))
	defer span.End()
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.PushPull(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to pushpull: %w", err)
	}
	return connect.NewResponse(res), nil

}
