package ratelimit

import (
	"context"
	"fmt"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
)

func ratelimitNodeKey(identifier string, limit int64, duration int64) string {
	window := time.Now().UnixMilli() / duration
	return fmt.Sprintf("ratelimit:%s:%d:%d", identifier, window, limit)
}

func (s *service) aggregateByOrigin(ctx context.Context, events []*ratelimitv1.PushPullEvent) {

	if len(events) == 0 {
		return
	}

	eventsByKey := map[string][]*ratelimitv1.PushPullEvent{}
	for _, e := range events {
		key := ratelimitNodeKey(e.Identifier, e.Limit, e.Duration)
		_, ok := eventsByKey[key]
		if !ok {
			eventsByKey[key] = []*ratelimitv1.PushPullEvent{}
		}
		eventsByKey[key] = append(eventsByKey[key], e)
	}

	for key, evts := range eventsByKey {
		s.syncBuffer <- syncWithOriginRequest{
			key:    key,
			events: evts,
		}
	}

}
