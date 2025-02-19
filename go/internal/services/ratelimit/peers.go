package ratelimit

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/tracing"
)

type peer struct {
	node   cluster.Node
	client ratelimitv1connect.RatelimitServiceClient
}

// syncs peers removes old peers based on the cluster's event listeners
//
// call this in a go routine
func (s *service) syncPeers() {
	for leave := range s.cluster.SubscribeLeave() {

		s.logger.Info(context.Background(), "peer left", slog.String("peer", leave.ID))
		s.peerMu.Lock()
		delete(s.peers, leave.ID)
		s.peerMu.Unlock()
	}

}

func (s *service) getPeer(key string) (peer, error) {

	s.peerMu.RLock()
	p, ok := s.peers[key]
	s.peerMu.RUnlock()
	if ok {
		return p, nil
	}

	p, err := s.newPeer(context.Background(), key)
	if err != nil {
		return peer{}, err
	}
	s.peerMu.Lock()
	s.peers[key] = p
	s.peerMu.Unlock()
	return p, nil

}

func (s *service) newPeer(ctx context.Context, key string) (peer, error) {
	ctx, span := tracing.Start(ctx, "ratelimit.newPeer")
	defer span.End()

	s.peerMu.Lock()
	defer s.peerMu.Unlock()

	node, err := s.cluster.FindNode(ctx, key)
	if err != nil {
		return peer{}, fault.Wrap(err, fault.WithDesc("failed to find node", "The ratelimit origin could not be found."))
	}

	s.logger.Info(ctx, "peer added", slog.String("peer", node.ID), slog.String("address", node.RpcAddr))
	rpcAddr := node.RpcAddr
	if !strings.Contains(rpcAddr, "://") {
		rpcAddr = "http://" + rpcAddr
	}

	interceptor, err := otelconnect.NewInterceptor(otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()))
	if err != nil {
		s.logger.Error(context.Background(), "failed to create interceptor", slog.String("error", err.Error()))
		return peer{}, err
	}

	c := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, rpcAddr, connect.WithInterceptors(interceptor))
	return peer{node: node, client: c}, nil
}
