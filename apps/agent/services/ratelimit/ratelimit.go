package ratelimit

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

func (s *service) Ratelimit(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {

	ctx, span := tracing.Start(ctx, "ratelimit.Ratelimit")
	defer span.End()

	ratelimitReq := ratelimit.RatelimitRequest{
		Name:       req.Name,
		Identifier: req.Identifier,
		Limit:      req.Limit,
		Duration:   time.Duration(req.Duration) * time.Millisecond,
		Cost:       req.Cost,
	}
	if req.Lease != nil {
		ratelimitReq.Lease = &ratelimit.Lease{
			Cost:      req.Lease.Cost,
			ExpiresAt: time.Now().Add(time.Duration(req.Lease.Timeout) * time.Millisecond),
		}
	}
	// TODO: reenable later
	if false && !s.ratelimiter.Has(ctx, ratelimitReq.Identifier, ratelimitReq.Duration) {
		originRes, err := s.ratelimitOrigin(ctx, req)
		if err != nil {
			s.logger.Err(err).Msg("failed to call ratelimit origin")
		}
		if originRes != nil {
			return originRes, nil
		}
	}

	taken := s.ratelimiter.Take(ctx, ratelimitReq)

	if s.batcher != nil {
		_, span := tracing.Start(ctx, "emitting pushPull event")
		span.SetAttributes(attribute.Int("channelSize", s.batcher.Size()))
		s.batcher.Buffer(&ratelimitv1.PushPullEvent{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
			Cost:       req.Cost,
			Time:       time.Now().UnixMilli(),
			Pass:       taken.Pass,
			Lease:      req.Lease,
		})

		span.End()

	}

	res := &ratelimitv1.RatelimitResponse{
		Current:   int64(taken.Current),
		Limit:     int64(taken.Limit),
		Remaining: int64(taken.Remaining),
		Reset_:    taken.Reset,
		Success:   taken.Pass,
	}

	if req.Lease != nil {
		res.Lease = &ratelimitv1.Lease{
			Identifier: req.Identifier,
			Limit:      req.Limit,
			Duration:   req.Duration,
		}
	}

	return res, nil

}

func (s *service) ratelimitOrigin(ctx context.Context, req *ratelimitv1.RatelimitRequest) (*ratelimitv1.RatelimitResponse, error) {
	ctx, span := tracing.Start(ctx, "ratelimit.RatelimitOrigin")
	defer span.End()

	s.logger.Info().Str("identifier", req.Identifier).Msg("no local state found, syncing with origin")
	key := ratelimitNodeKey(req.Identifier, req.Duration)
	peer, err := s.cluster.FindNode(key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Str("key", key).Msg("unable to find responsible nodes")
		return nil, err
	}

	if peer.Id == s.cluster.NodeId() {
		return nil, nil
	}

	s.consistencyChecker.Record(key, peer.Id)

	url := peer.RpcAddr
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}

	s.peersMu.RLock()
	c, ok := s.peers[url]
	s.peersMu.RUnlock()
	if !ok {
		interceptor, err := otelconnect.NewInterceptor(otelconnect.WithTracerProvider(tracing.GetGlobalTraceProvider()))
		if err != nil {
			tracing.RecordError(span, err)
			s.logger.Err(err).Msg("failed to create interceptor")
			return nil, err
		}
		c = ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, url, connect.WithInterceptors(interceptor))
		s.peersMu.Lock()
		s.peers[url] = c
		s.peersMu.Unlock()
	}

	connectReq := connect.NewRequest(req)

	connectReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", s.cluster.AuthToken()))

	res, err := c.Ratelimit(ctx, connectReq)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Err(err).Msg("failed to call ratelimit")
		return nil, err
	}

	s.ratelimiter.SetCurrent(ctx, ratelimit.SetCurrentRequest{
		Identifier: req.Identifier,
		Limit:      req.Limit,
		Duration:   time.Duration(req.Duration) * time.Millisecond,
		Current:    res.Msg.Current,
	})

	return res.Msg, nil

}
