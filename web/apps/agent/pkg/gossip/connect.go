package gossip

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	gossipv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/gossip/v1"
	"github.com/unkeyed/unkey/svc/agent/gen/proto/gossip/v1/gossipv1connect"
	"github.com/unkeyed/unkey/svc/agent/pkg/logging"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type clusterServer struct {
	svc    *cluster
	logger logging.Logger
	close  chan struct{}
	gossipv1connect.UnimplementedGossipServiceHandler
}

func NewClusterServer(svc *cluster, logger logging.Logger) *clusterServer {

	return &clusterServer{
		svc:    svc,
		logger: logger,
		close:  make(chan struct{}),
	}
}

func (s *clusterServer) CreateHandler() (string, http.Handler, error) {
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return "", nil, err
	}

	path, handler := gossipv1connect.NewGossipServiceHandler(s, connect.WithInterceptors(otelInterceptor))
	return path, handler, nil
}

func (c *clusterServer) Serve() error {

	mux := http.NewServeMux()

	path, handler, err := c.CreateHandler()
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to create handler"))
	}
	mux.Handle(path, handler)

	u, err := url.Parse(c.svc.self.RpcAddr)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to parse self rpc addr"))
	}

	srv := &http.Server{Addr: u.Host, Handler: h2c.NewHandler(mux, &http2.Server{})}

	c.logger.Info().Str("addr", u.Host).Msg("listening")

	go func() {
		<-c.close
		_ = srv.Close()
	}()

	return srv.ListenAndServe()
}

func (s *clusterServer) Join(
	ctx context.Context,
	req *connect.Request[gossipv1.JoinRequest],
) (*connect.Response[gossipv1.JoinResponse], error) {

	res, err := s.svc.join(ctx, req.Msg)

	return connect.NewResponse(res), err
}

func (s *clusterServer) Leave(
	ctx context.Context,
	req *connect.Request[gossipv1.LeaveRequest],
) (*connect.Response[gossipv1.LeaveResponse], error) {

	res, err := s.svc.leave(ctx, req.Msg)

	return connect.NewResponse(res), err
}

func (s *clusterServer) Ping(
	ctx context.Context,
	req *connect.Request[gossipv1.PingRequest],
) (*connect.Response[gossipv1.PingResponse], error) {

	res, err := s.svc.ping(ctx, req.Msg)

	return connect.NewResponse(res), err
}

func (s *clusterServer) IndirectPing(
	ctx context.Context,
	req *connect.Request[gossipv1.IndirectPingRequest],
) (*connect.Response[gossipv1.IndirectPingResponse], error) {

	res, err := s.svc.indirectPing(ctx, req.Msg)

	return connect.NewResponse(res), err
}

func (s *clusterServer) SyncMembers(
	ctx context.Context,
	req *connect.Request[gossipv1.SyncMembersRequest],
) (*connect.Response[gossipv1.SyncMembersResponse], error) {

	res, err := s.svc.syncMembers(ctx, req.Msg)

	return connect.NewResponse(res), err
}
