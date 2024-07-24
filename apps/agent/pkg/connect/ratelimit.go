package connect

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/apps/agent/services/ratelimit"
)

var _ ratelimitv1connect.RatelimitServiceHandler = (*ratelimitServer)(nil)

type ratelimitServer struct {
	svc       ratelimit.Service
	logger    logging.Logger
	authToken string
	ratelimitv1connect.UnimplementedRatelimitServiceHandler
}

func NewRatelimitServer(svc ratelimit.Service, logger logging.Logger, authToken string) *ratelimitServer {

	return &ratelimitServer{
		svc:       svc,
		logger:    logger,
		authToken: authToken,
	}

}

func (s *ratelimitServer) CreateHandler() (string, http.Handler, error) {
	otelInterceptor, err := otelconnect.NewInterceptor(otelconnect.WithTraceRequestHeader("Unkey-Request-Id"))
	if err != nil {
		return "", nil, err
	}

	path, handler := ratelimitv1connect.NewRatelimitServiceHandler(s, connect.WithInterceptors(otelInterceptor))
	return path, handler, nil

}
func (s *ratelimitServer) Ratelimit(
	ctx context.Context,
	req *connect.Request[ratelimitv1.RatelimitRequest],
) (*connect.Response[ratelimitv1.RatelimitResponse], error) {

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "Ratelimit"))
	defer span.End()
	err := auth.Authorize(ctx, s.authToken, req.Header().Get("Authorization"))
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

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "MultiRatelimit"))
	defer span.End()
	err := auth.Authorize(ctx, s.authToken, req.Header().Get("Authorization"))
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

func (s *ratelimitServer) Liveness(
	ctx context.Context,
	req *connect.Request[ratelimitv1.LivenessRequest],
) (*connect.Response[ratelimitv1.LivenessResponse], error) {

	return connect.NewResponse(&ratelimitv1.LivenessResponse{
		Status: "ok",
	}), nil

}

func (s *ratelimitServer) PushPull(
	ctx context.Context,
	req *connect.Request[ratelimitv1.PushPullRequest],
) (*connect.Response[ratelimitv1.PushPullResponse], error) {

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "PushPull"))
	defer span.End()
	err := auth.Authorize(ctx, s.authToken, req.Header().Get("Authorization"))
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
