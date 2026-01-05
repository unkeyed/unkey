package connect

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/svc/agent/pkg/auth"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
)

var _ ratelimitv1connect.RatelimitServiceHandler = (*ratelimitServer)(nil)

type RatelimitService interface {
	Ratelimit(context.Context, *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error)
	MultiRatelimit(context.Context, *ratelimitv1.RatelimitMultiRequest) (*ratelimitv1.RatelimitMultiResponse, error)
	PushPull(context.Context, *ratelimitv1.PushPullRequest) (*ratelimitv1.PushPullResponse, error)
	CommitLease(context.Context, *ratelimitv1.CommitLeaseRequest) (*ratelimitv1.CommitLeaseResponse, error)
	Mitigate(context.Context, *ratelimitv1.MitigateRequest) (*ratelimitv1.MitigateResponse, error)
}
type ratelimitServer struct {
	svc       RatelimitService
	logger    logging.Logger
	authToken string
	ratelimitv1connect.UnimplementedRatelimitServiceHandler
}

func NewRatelimitServer(svc RatelimitService, logger logging.Logger, authToken string) *ratelimitServer {

	return &ratelimitServer{
		svc:       svc,
		logger:    logger,
		authToken: authToken,
	}

}

func (s *ratelimitServer) CreateHandler() (string, http.Handler, error) {
	otelInterceptor, err := otelconnect.NewInterceptor()
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

func (s *ratelimitServer) Mitigate(
	ctx context.Context,
	req *connect.Request[ratelimitv1.MitigateRequest],
) (*connect.Response[ratelimitv1.MitigateResponse], error) {

	ctx, span := tracing.Start(ctx, tracing.NewSpanName("connect.ratelimit", "Mitigate"))
	defer span.End()
	err := auth.Authorize(ctx, s.authToken, req.Header().Get("Authorization"))
	if err != nil {

		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	res, err := s.svc.Mitigate(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to pushpull: %w", err)
	}
	return connect.NewResponse(res), nil

}
