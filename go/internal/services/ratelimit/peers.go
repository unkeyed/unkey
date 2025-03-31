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

// peer represents a node in the rate limiting cluster that can handle
// rate limit requests. It maintains a connection to the remote node
// and provides methods for state synchronization.
//
// Thread Safety:
//   - Immutable after creation
//   - Safe for concurrent use
//
// Performance:
//   - Connection pooling handled by HTTP client
//   - Minimal memory footprint per peer
type peer struct {
	// instance contains cluster metadata about the peer
	instance cluster.Instance

	// client is the RPC client for communicating with the peer
	// Thread-safe for concurrent use
	client ratelimitv1connect.RatelimitServiceClient
}

// syncPeers maintains the service's peer list by listening for cluster membership
// changes and removing peers that have left the cluster. This ensures the rate
// limiter only attempts to communicate with active cluster nodes.
//
// This method should be run in a separate goroutine as it blocks while listening
// for cluster events.
//
// Thread Safety:
//   - Safe for concurrent access with other peer operations
//   - Uses peerMu to protect peer list modifications
//
// Performance:
//   - O(1) per peer removal
//   - Minimal memory usage
//   - Non-blocking for rate limit operations
//
// Example Usage:
//
//	go service.syncPeers() // Start peer synchronization
func (s *service) syncPeers() {
	for leave := range s.cluster.SubscribeLeave() {

		s.logger.Info("peer left", "peer", leave.ID)
		s.peerMu.Lock()
		delete(s.peers, leave.ID)
		s.peerMu.Unlock()
	}

}

// getPeer retrieves or creates a peer connection for the given key.
// The key is used with consistent hashing to determine which node
// should be the origin for a particular rate limit identifier.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - key: Consistent hash key to identify the peer
//
// Returns:
//   - peer: The peer connection, either existing or newly created
//   - error: Any errors during peer lookup or connection
//
// Thread Safety:
//   - Safe for concurrent use
//   - Uses read-write mutex for peer map access
//
// Performance:
//   - O(1) for existing peers
//   - Network round trip for new peer connections
//   - Connection pooling reduces overhead
//
// Errors:
//   - Returns error if peer instance cannot be found
//   - Returns error if connection cannot be established
//
// Example:
//
//	p, err := svc.getPeer(ctx, "user-123")
//	if err != nil {
//	    return fmt.Errorf("failed to get peer: %w", err)
//	}
//	resp, err := p.client.Ratelimit(ctx, req)
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

// newPeer creates a new peer connection to a cluster node.
// It establishes the RPC client connection and configures tracing.
//
// Parameters:
//   - ctx: Context for cancellation and tracing
//   - key: Consistent hash key to identify the peer
//
// Returns:
//   - peer: Newly created peer connection
//   - error: Any errors during peer creation
//
// Thread Safety:
//   - Caller must hold peerMu lock
//   - Resulting peer is safe for concurrent use
//
// Performance:
//   - Network round trip for initial connection
//   - Creates new HTTP client and interceptors
//
// Errors:
//   - Returns error if instance lookup fails
//   - Returns error if interceptor creation fails
//   - Returns error if RPC client creation fails
//
// Example:
//
//	s.peerMu.Lock()
//	p, err := s.newPeer(ctx, "user-123")
//	if err != nil {
//	    s.peerMu.Unlock()
//	    return fmt.Errorf("failed to create peer: %w", err)
//	}
//	s.peers[key] = p
//	s.peerMu.Unlock()
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
