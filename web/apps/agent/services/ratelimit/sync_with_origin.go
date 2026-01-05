package ratelimit

import (
	"context"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/svc/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/svc/agent/pkg/prometheus"
	"github.com/unkeyed/unkey/svc/agent/pkg/tracing"
)

type syncWithOriginRequest struct {
	req         *ratelimitv1.PushPullRequest
	localPassed bool
}

func (s *service) syncWithOrigin(req syncWithOriginRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx, span := tracing.Start(ctx, "ratelimit.syncWithOrigin")
	defer span.End()

	t := time.UnixMilli(req.req.Time)
	duration := time.Duration(req.req.Request.Duration) * time.Millisecond

	key := bucketKey{req.req.Request.Identifier, req.req.Request.Limit, duration}.toString()
	client, peer, err := s.getPeerClient(ctx, key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Str("key", key).Msg("unable to create peer client")
		return
	}
	if peer.Id == s.cluster.NodeId() {
		return
	}

	res, err := s.syncCircuitBreaker.Do(ctx, func(innerCtx context.Context) (*connect.Response[ratelimitv1.PushPullResponse], error) {
		innerCtx, cancel = context.WithTimeout(innerCtx, 10*time.Second)
		defer cancel()
		return client.PushPull(innerCtx, connect.NewRequest(req.req))
	})
	if err != nil {
		s.peersMu.Lock()
		s.logger.Warn().Str("peerId", peer.Id).Err(err).Msg("resetting peer client due to error")
		delete(s.peers, peer.Id)
		s.peersMu.Unlock()
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Str("peerId", peer.Id).Str("addr", peer.RpcAddr).Msg("failed to push pull")
		return
	}

	err = s.SetCounter(ctx,
		setCounterRequest{
			Identifier: req.req.Request.Identifier,
			Limit:      req.req.Request.Limit,
			Counter:    res.Msg.Current.Counter,
			Sequence:   res.Msg.Current.Sequence,
			Duration:   duration,
			Time:       t,
		},
		setCounterRequest{
			Identifier: req.req.Request.Identifier,

			Counter:  res.Msg.Previous.Counter,
			Sequence: res.Msg.Previous.Sequence,
			Duration: duration,
			Time:     t,
		},
	)

	if req.localPassed == res.Msg.Response.Success {
		ratelimitAccuracy.WithLabelValues("true").Inc()
	} else {
		ratelimitAccuracy.WithLabelValues("false").Inc()
	}

	// req.events is guaranteed to have at least element
	// and the first one should be the oldest event, so we can use it to get the max latency
	latency := time.Since(t)
	labels := map[string]string{
		"nodeId": s.cluster.NodeId(),
		"peerId": peer.Id,
	}
	prometheus.RatelimitPushPullEvents.With(labels).Inc()

	prometheus.RatelimitPushPullLatency.With(labels).Observe(latency.Seconds())

	// if we got this far, we pushpulled successfully with a peer and don't need to try the rest

}
