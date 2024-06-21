package connect

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bufbuild/connect-go"
	clusterv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1/clusterv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type clusterServer struct {
	svc    *cluster.Cluster
	logger logging.Logger
	clusterv1connect.UnimplementedRatelimitServiceHandler
}

func NewClusterServer(svc *cluster.Cluster, logger logging.Logger) *clusterServer {

	return &clusterServer{
		svc:    svc,
		logger: logger,
	}
}

func (s *clusterServer) CreateHandler() (string, http.Handler) {
	return clusterv1connect.NewRatelimitServiceHandler(s)

}

func (s *clusterServer) Join(
	ctx context.Context,
	req *connect.Request[clusterv1.JoinClusterRequest],
) (*connect.Response[clusterv1.JoinClusterResponse], error) {
	// authorization := req.Header().Get("Authorization")
	// err := auth.Authorize(ctx, authorization)
	// if err != nil {
	// 	s.logger.Warn().Err(err).Msg("failed to authorize request")
	// 	return nil, err
	// }

	size, err := s.svc.Join(req.Msg.Addrs)
	if err != nil {
		return nil, fmt.Errorf("failed to join cluster: %w", err)
	}
	return connect.NewResponse(&clusterv1.JoinClusterResponse{Size: int32(size)}), nil

}

func (s *clusterServer) Leave(
	ctx context.Context,
	req *connect.Request[clusterv1.LeaveClusterRequest],
) (*connect.Response[clusterv1.LeaveClusterResponse], error) {
	// authorization := req.Header().Get("Authorization")
	// err := auth.Authorize(ctx, authorization)
	// if err != nil {
	// 	s.logger.Warn().Err(err).Msg("failed to authorize request")
	// 	return nil, err
	// }

	err := s.svc.Shutdown()
	if err != nil {
		return nil, fmt.Errorf("failed to leave cluster: %w", err)
	}
	return connect.NewResponse(&clusterv1.LeaveClusterResponse{}), nil

}
