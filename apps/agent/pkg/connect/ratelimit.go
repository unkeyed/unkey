package connect

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/ratelimit"
)

type ratelimitServer struct {
	svc *ratelimit.Service
	ratelimitv1connect.UnimplementedRatelimitServiceHandler
}

func NewRatelimitServer(svc *ratelimit.Service) *ratelimitServer {

	return &ratelimitServer{
		svc: svc,
	}
}

func (s *ratelimitServer) CreateHandler() (string, http.Handler) {
	return ratelimitv1connect.NewRatelimitServiceHandler(s)

}

func (s *ratelimitServer) Ratelimit(
	ctx context.Context,
	req *connect.Request[ratelimitv1.RatelimitRequest],
) (*connect.Response[ratelimitv1.RatelimitResponse], error) {
	err := auth.Authorize(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	
	res, err := s.svc.Ratelimit(ctx, req.Msg)
	if err != nil {
		return nil, fmt.Errorf("failed to ratelimit: %w", err)
	}
	return connect.NewResponse(res), nil

}
