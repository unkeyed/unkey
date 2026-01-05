package ratelimit

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
)

// Mitigate is an RPC handler receiving a mitigation broadcast from the origin node
// and applying the mitigation to the local sliding window
func (s *service) Mitigate(ctx context.Context, req *ratelimitv1.MitigateRequest) (*ratelimitv1.MitigateResponse, error) {
	ctx, span := tracing.Start(ctx, "ratelimit.Mitigate")
	defer span.End()

	s.logger.Info().Interface("req", req).Msg("mitigating")

	duration := time.Duration(req.Duration) * time.Millisecond
	bucket, _ := s.getBucket(bucketKey{req.Identifier, req.Limit, duration})
	bucket.Lock()
	defer bucket.Unlock()
	bucket.windows[req.Window.GetSequence()] = req.Window

	return &ratelimitv1.MitigateResponse{}, nil
}

type mitigateWindowRequest struct {
	identifier string
	limit      int64
	duration   time.Duration
	window     *ratelimitv1.Window
}

func (s *service) broadcastMitigation(req mitigateWindowRequest) {
	ctx := context.Background()
	node, err := s.cluster.FindNode(bucketKey{req.identifier, req.limit, req.duration}.toString())
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to find node")
		return
	}
	if node.Id != s.cluster.NodeId() {
		return
	}

	peers, err := s.getAllPeers(ctx)
	if err != nil {
		s.logger.Err(err).Msg("failed to get peers")
		return
	}
	for _, peer := range peers {
		_, err := s.mitigateCircuitBreaker.Do(ctx, func(innerCtx context.Context) (*connect.Response[ratelimitv1.MitigateResponse], error) {
			innerCtx, cancel := context.WithTimeout(innerCtx, 10*time.Second)
			defer cancel()
			return peer.client.Mitigate(innerCtx, connect.NewRequest(&ratelimitv1.MitigateRequest{
				Identifier: req.identifier,
				Limit:      req.limit,
				Duration:   req.duration.Milliseconds(),
				Window:     req.window,
			}))
		})
		if err != nil {
			s.logger.Err(err).Msg("failed to call mitigate")
		} else {
			s.logger.Debug().Str("peerId", peer.id).Msg("broadcasted mitigation")
		}
	}
}
