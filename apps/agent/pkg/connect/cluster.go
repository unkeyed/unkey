package connect

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
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

func (s *clusterServer) CreateHandler() (string, http.Handler, error) {
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return "", nil, err
	}

	path, handler := clusterv1connect.NewClusterServiceHandler(s, connect.WithInterceptors(otelInterceptor))
	return path, handler, nil

}

func (s *clusterServer) AnnounceStateChange(
	ctx context.Context,
	req *connect.Request[clusterv1.AnnounceStateChangeRequest],
) (*connect.Response[clusterv1.AnnounceStateChangeResponse], error) {
	authorization := req.Header().Get("Authorization")
	err := auth.Authorize(ctx, "TODO:", authorization)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to authorize request")
		return nil, err
	}

	return connect.NewResponse(&clusterv1.AnnounceStateChangeResponse{}), nil
}
