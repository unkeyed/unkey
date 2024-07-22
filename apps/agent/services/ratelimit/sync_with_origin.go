package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1/ratelimitv1connect"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
)

type syncWithOriginRequest struct {
	key    string
	events []*ratelimitv1.PushPullEvent
}

func (s *service) syncWithOrigin(req syncWithOriginRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx, span := tracing.Start(ctx, "ratelimit.syncWithOrigin")
	defer span.End()

	peer, err := s.cluster.FindNode(req.key)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Err(err).Str("key", req.key).Msg("unable to find responsible nodes")
		return
	}

	if peer.Id == s.cluster.NodeId() {
		s.logger.Debug().Str("key", req.key).Msg("skipping push pull with self")
		return
	}

	s.consistencyChecker.Record(req.key, peer.Id)

	s.logger.Debug().Str("peerId", peer.Id).Str("key", req.key).Int("events", len(req.events)).Msg("push pull with")

	url := peer.RpcAddr
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}
	c := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, url)

	connectReq := connect.NewRequest(&ratelimitv1.PushPullRequest{
		Events: req.events,
	})
	s.logger.Info().Interface("req", connectReq).Msg("push pull request")
	connectReq.Header().Set("Authorization", fmt.Sprintf("Bearer %s", s.cluster.AuthToken()))

	res, err := c.PushPull(ctx, connectReq)
	if err != nil {
		tracing.RecordError(span, err)
		s.logger.Warn().Interface("req headers", connectReq.Header().Clone()).Err(err).Interface("peer", peer).Str("peerId", peer.Id).Msg("failed to push pull")
		return
	}
	s.logger.Debug().Str("peerId", peer.Id).Str("key", req.key).Interface("res", res).Msg("push pull came back")

	if len(req.events) != len(res.Msg.Updates) {
		s.logger.Error().Msg("length of updates does not match length of events, unable to set current")
		return
	}
	for i, e := range req.events {
		err := s.ratelimiter.SetCurrent(ctx, ratelimit.SetCurrentRequest{
			Identifier:     e.Identifier,
			Max:            e.Limit,
			Current:        res.Msg.Updates[i].Current,
			RefillInterval: e.Duration,
		})
		if err != nil {
			tracing.RecordError(span, err)
			s.logger.Error().Err(err).Msg("failed to set current")
			return
		}
	}
	// if we got this far, we pushpulled successfully with a peer and don't need to try the rest

}
