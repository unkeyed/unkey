package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/go/pkg/tracing"
)

func (s *service) getPeer(peerID string) (ratelimitv1connect.RatelimitServiceClient, error) {
	s.peerMu.RLock()
	defer s.peerMu.RUnlock()

	peer, ok := s.peers[peerID]
	if !ok {
		return nil, fmt.Errorf("peer not found")
	}
	return peer, nil

}

func (s *service) newPeer(id string, rpcAddr string) (ratelimitv1connect.RatelimitServiceClient, error) {

	if !strings.Contains(rpcAddr, "://") {
		rpcAddr = "http://" + rpcAddr
	}

	interceptor, err := otelconnect.NewInterceptor(otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()))
	if err != nil {
		s.logger.Error(context.Background(), "failed to create interceptor", slog.String("error", err.Error()))
		return nil, err
	}

	c := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, rpcAddr, connect.WithInterceptors(interceptor))
	return c, nil
}
