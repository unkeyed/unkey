package connect

import (
	"context"
	"net/http"

	"github.com/bufbuild/connect-go"
	clusterv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/cluster/v1/clusterv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/auth"
	"github.com/unkeyed/unkey/apps/agent/pkg/cluster"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
)

type clusterServer struct {
	svc    cluster.Cluster
	logger logging.Logger
	clusterv1connect.UnimplementedClusterServiceHandler
}

func NewClusterServer(svc cluster.Cluster, logger logging.Logger) *clusterServer {

	return &clusterServer{
		svc:    svc,
		logger: logger,
	}
}

func (s *clusterServer) CreateHandler() (string, http.Handler) {
	return clusterv1connect.NewClusterServiceHandler(s)

}

func (s *clusterServer) AnnounceStateChange(
	ctx context.Context,
	req *connect.Request[clusterv1.AnnounceStateChangeRequest],
) (*connect.Response[clusterv1.AnnounceStateChangeResponse], error) {
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	err = s.svc.SyncMembership()
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to sync membership")
		return nil, err
	}
	return connect.NewResponse(&clusterv1.AnnounceStateChangeResponse{}), nil
}
