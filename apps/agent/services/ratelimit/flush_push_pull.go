package ratelimit

import (
	"context"
	"time"

	ratelimitv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/ratelimit/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/ratelimit"
)

func (s *service) aggregateByOrigin(ctx context.Context, events []*ratelimitv1.PushPullEvent) {

	if len(events) == 0 {
		return
	}

	eventsByKey := map[string][]*ratelimitv1.PushPullEvent{}
	for _, e := range events {
		key := ratelimit.BuildKey(e.Identifier, time.Duration(e.Duration)*time.Millisecond)
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
