package ratelimit

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/go/pkg/otel/metrics"
	"github.com/unkeyed/unkey/go/pkg/repeat"
)

func (s *service) expireWindowsAndBuckets() {

	repeat.Every(time.Minute, func() {
		ctx := context.Background()
		s.bucketsMu.Lock()
		defer s.bucketsMu.Unlock()

		windows := int64(0)

		for bucketID, bucket := range s.buckets {
			bucket.mu.Lock()
			for sequence, window := range bucket.windows {
				if s.clock.Now().UnixMilli() > (window.Start + (3 * window.Duration)) {
					delete(bucket.windows, sequence)
					metrics.Ratelimit.EvictedWindows.Add(ctx, 1)
				} else {
					windows++
				}
			}
			if len(bucket.windows) == 0 {
				delete(s.buckets, bucketID)
				metrics.Ratelimit.EvictedBuckets.Add(ctx, 1)
			}

			bucket.mu.Unlock()
		}

		metrics.Ratelimit.Buckets.Record(ctx, int64(len(s.buckets)))
		metrics.Ratelimit.Windows.Record(ctx, windows)
	})

}
