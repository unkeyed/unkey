package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

func (s *service) CommitLease(ctx context.Context, req *ratelimitv1.CommitLeaseRequest) (*ratelimitv1.CommitLeaseResponse, error) {
	ctx, span := tracing.Start(ctx, "svc.ratelimit.CommitLease")
	defer span.End()

	key := ratelimitNodeKey(req.Lease.Identifier, req.Lease.Limit, req.Lease.Duration)

	origin, err := s.cluster.FindNode(key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Str("key", key).Msg("unable to find responsible nodes")
		return nil, nil
	}

	// If we're the origin, we can commit the lease locally and return
	if origin.Id == s.cluster.NodeId() {
		s.ratelimiter.CommitLease(ctx, ratelimit.CommitLeaseRequest{
			Identifier: "TODO",
			LeaseId:    "TODO",
			Tokens:     req.Cost,
		})

		return &ratelimitv1.CommitLeaseResponse{}, nil
	}

	// Else we need to forward the request to the responsible node

	url := origin.RpcAddr
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
			return nil, fault.Wrap(err, fmsg.With("failed to create interceptor"))
		}
		c = ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, url, connect.WithInterceptors(interceptor))
		s.peersMu.Lock()
		s.peers[url] = c
		s.peersMu.Unlock()
	}

	connectReq := connect.NewRequest(req)

	connectReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", s.cluster.AuthToken()))

	res, err := c.CommitLease(ctx, connectReq)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Err(err).Msg("failed to commit lease")
		return nil, fault.Wrap(err)
	}
	return res.Msg, nil

}
