package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/uid"
)

func TestTakeCreatesWindows(t *testing.T) {
	rl, err := New(Config{
		Logger:  logging.NewNoopLogger(),
		Metrics: metrics.NewNoop(),
	})
	require.NoError(t, err)

	now := time.Now()

	identifier := "test"
	limit := int64(10)
	duration := time.Minute

	res := rl.Take(context.Background(), ratelimitRequest{
		Time:       now,
		Name:       "test",
		Identifier: identifier,
		Limit:      limit,
		Duration:   duration,
		Cost:       1,
	})

	require.Equal(t, int64(10), res.Limit)
	require.Equal(t, int64(9), res.Remaining)
	require.Equal(t, int64(1), res.Current)
	require.True(t, res.Pass)
	require.Equal(t, int64(0), res.previousWindow.Counter)
	require.Equal(t, int64(1), res.currentWindow.Counter)

	rl.bucketsMu.RLock()
	bucket, ok := rl.buckets[bucketKey{identifier, limit, duration}.toString()]
	rl.bucketsMu.RUnlock()
	require.True(t, ok)

	bucket.Lock()
	sequence := now.UnixMilli() / duration.Milliseconds()
	currentWindow, ok := bucket.windows[sequence]
	require.True(t, ok)
	require.Equal(t, int64(1), currentWindow.Counter)

	previousWindow, ok := bucket.windows[sequence-1]
	require.True(t, ok)
	require.Equal(t, int64(0), previousWindow.Counter)

}

func TestSlidingWindowAccuracy(t *testing.T) {
	rl, err := New(Config{
		Logger:  logging.New(nil),
		Metrics: metrics.NewNoop(),
	})
	require.NoError(t, err)

	for _, limit := range []int64{
		5,
		10,
		100,
		500,
	} {
		for _, duration := range []time.Duration{
			1 * time.Second,
			10 * time.Second,
			1 * time.Minute,
			5 * time.Minute,
			1 * time.Hour,
		} {
			for _, windows := range []int64{1, 2, 5, 10, 50} {
				requests := limit * windows * 1000
				t.Run(fmt.Sprintf("rate %d/%s  %d requests across %d windows",
					limit,
					duration,
					requests,
					windows,
				), func(t *testing.T) {

					identifier := uid.New("test")

					passed := int64(0)
					total := time.Duration(windows) * duration
					dt := total / time.Duration(requests)
					now := time.Now().Truncate(duration)
					for i := int64(0); i < requests; i++ {
						res := rl.Take(context.Background(), ratelimitRequest{
							Time:       now.Add(time.Duration(i) * dt),
							Identifier: identifier,
							Limit:      limit,
							Duration:   duration,
							Cost:       1,
						})
						if res.Pass {
							passed++
						}
					}

					require.GreaterOrEqual(t, passed, int64(float64(limit)*float64(windows)*0.8), "%d out of %d passed", passed, requests)
					require.LessOrEqual(t, passed, limit*(windows+1), "%d out of %d passed", passed, requests)

				})
			}

		}
	}

}
