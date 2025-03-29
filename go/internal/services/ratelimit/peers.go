package ratelimit

import (
	"context"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"github.com/unkeyed/unkey/go/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/go/pkg/cluster"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type peer struct {
	instance cluster.Instance
	client   ratelimitv1connect.RatelimitServiceClient
}

// syncs peers removes old peers based on the cluster's event listeners
//
// call this in a go routine
func (s *service) syncPeers() {
	for leave := range s.cluster.SubscribeLeave() {

		s.logger.Info("peer left", "peer", leave.ID)
		s.peerMu.Lock()
		delete(s.peers, leave.ID)
		s.peerMu.Unlock()
	}

}

func (s *service) getPeer(ctx context.Context, key string) (peer, error) {
	ctx, span := tracing.Start(ctx, "getPeer")
	defer span.End()

	var p peer

	defer func() {
		metrics.Ratelimit.Origin.Add(ctx, 1, metric.WithAttributeSet(attribute.NewSet(
			attribute.String("origin_instance_id", p.instance.ID),
		),
		))
	}()

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

	instance, err := s.cluster.FindInstance(ctx, key)
	if err != nil {
		return peer{}, fault.Wrap(err, fault.WithDesc("failed to find instance", "The ratelimit origin could not be found."))
	}

	s.logger.Info("peer added",
		"peer", instance.ID,
		"address", instance.RpcAddr,
	)
	rpcAddr := instance.RpcAddr
	if !strings.Contains(rpcAddr, "://") {
		rpcAddr = "http://" + rpcAddr
	}

	interceptor, err := otelconnect.NewInterceptor(
		otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()),
		otelconnect.WithoutServerPeerAttributes(),
	)
	if err != nil {
		s.logger.Error("failed to create interceptor", "error", err.Error())
		return peer{}, err
	}

	c := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, rpcAddr, connect.WithInterceptors(interceptor))
	return peer{instance: instance, client: c}, nil
}
