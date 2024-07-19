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
)

func ratelimitNodeKey(identifier string, limit int64, duration int64) string {
	window := time.Now().UnixMilli() / duration
	return fmt.Sprintf("ratelimit:%s:%d:%d", identifier, window, limit)
}

func (s *service) flushPushPull(ctx context.Context, events []*ratelimitv1.PushPullEvent) {

	if len(events) == 0 {
		return
	}

	s.logger.Info().Int("events", len(events)).Msg("Flushing")
	eventsByKey := map[string][]*ratelimitv1.PushPullEvent{}
	for _, e := range events {
		key := ratelimitNodeKey(e.Identifier, e.Limit, e.Duration)
		if _, ok := eventsByKey[key]; !ok {
			eventsByKey[key] = []*ratelimitv1.PushPullEvent{}
		}
		eventsByKey[key] = append(eventsByKey[key], e)
	}

	for key, evts := range eventsByKey {
		go s.sync(ctx, key, evts)
	}
}

func (s *service) sync(ctx context.Context, key string, events []*ratelimitv1.PushPullEvent) {
	peer, err := s.cluster.FindNode(key)
	if err != nil {
		s.logger.Warn().Err(err).Str("key", key).Msg("unable to find responsible nodes")
		return
	}

	if peer.Id == s.cluster.NodeId() {
		s.logger.Debug().Str("key", key).Msg("skipping push pull with self")
		return
	}

	s.metrics.Record(key, peer.Id)

	s.logger.Info().Str("peerId", peer.Id).Str("key", key).Int("events", len(events)).Msg("push pull with")

	url := peer.RpcAddr
	if !strings.Contains(url, "://") {
		url = "http://" + url
	}

	c := ratelimitv1connect.NewRatelimitServiceClient(http.DefaultClient, url)

	req := connect.NewRequest(&ratelimitv1.PushPullRequest{
		Events: events,
	})
	s.logger.Info().Interface("req", req).Msg("push pull request")
	req.Header().Set("Authorization", fmt.Sprintf("Bearer %s", s.cluster.AuthToken()))

	res, err := c.PushPull(ctx, req)
	if err != nil {
		s.logger.Warn().Interface("req headers", req.Header().Clone()).Err(err).Interface("peer", peer).Str("peerId", peer.Id).Msg("failed to push pull")
		return
	}
	s.logger.Debug().Str("peerId", peer.Id).Str("key", key).Interface("res", res).Msg("push pull came back")

	if len(events) != len(res.Msg.Updates) {
		s.logger.Error().Msg("length of updates does not match length of events, unable to set current")
		return
	}

	for i, e := range events {
		err = s.ratelimiter.SetCurrent(ctx, ratelimit.SetCurrentRequest{
			Identifier:     e.Identifier,
			Max:            e.Limit,
			Current:        res.Msg.Updates[i].Current,
			RefillInterval: e.Duration,
		})
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to set current")
			return
		}
	}
	// if we got this far, we pushpulled successfully with a peer and don't need to try the rest

}
