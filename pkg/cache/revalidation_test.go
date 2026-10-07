package cache

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var testMetrics = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(testMetrics)
	os.Exit(m.Run())
}

func TestStaleReadsDoNotWaitForRefreshQueue(t *testing.T) {
	for _, method := range []string{"SWR", "SWRMany", "SWRWithFallback"} {
		t.Run(method, func(t *testing.T) {
			c, clk := newPausedRefreshCache(t)
			c.Set(t.Context(), "target", "stale")
			clk.Tick(2 * time.Minute)
			for range cap(c.revalidateC) {
				c.revalidateC <- func() {}
			}

			type result struct {
				value string
				hit   CacheHit
				err   error
			}
			done := make(chan result, 1)
			returned := make(chan struct{})
			go func() {
				defer close(returned)
				value, hit, err := readStale(c, method, "target")
				done <- result{value, hit, err}
			}()
			t.Cleanup(func() {
				c.Close()
				<-returned
			})

			select {
			case got := <-done:
				require.NoError(t, got.err)
				require.Equal(t, "stale", got.value)
				require.Equal(t, Hit, got.hit)
			case <-time.After(time.Second):
				t.Fatal("stale read blocked on a full refresh queue")
			}

			for len(c.revalidateC) > 0 {
				<-c.revalidateC
			}
			_, _, err := readStale(c, method, "target")
			require.NoError(t, err)
			require.Len(t, c.revalidateC, 1, "a dropped refresh must be retryable")
			(<-c.revalidateC)()
			value, hit := c.Get(t.Context(), "target")
			require.Equal(t, Hit, hit)
			require.Equal(t, "refreshed-target", value)

			c.Close()
			clk.Tick(2 * time.Minute)
			value, hit, err = readStale(c, method, "target")
			require.NoError(t, err)
			require.Equal(t, Hit, hit)
			require.Equal(t, "refreshed-target", value)
			require.Empty(t, c.revalidateC)
			require.Empty(t, c.inflightRefreshes)

			for _, outcome := range []string{"enqueued", "queue_full", "closed"} {
				metric := gatherCacheMetric(t, "unkey_cache_revalidation_enqueues_total", c.resource, outcome)
				require.Equal(t, float64(1), metric.GetCounter().GetValue(), outcome)
			}
			depth := gatherCacheMetric(t, "unkey_cache_revalidation_queue_depth", c.resource, "").GetHistogram()
			require.Equal(t, uint64(2), depth.GetSampleCount())
			require.Equal(t, float64(1000), depth.GetSampleSum(), "queue pressure must include dropped attempts")
			require.Equal(t, uint64(1), depth.GetBucket()[0].GetCumulativeCount(), "only the retry saw an empty queue")
			wait := gatherCacheMetric(t, "unkey_cache_revalidation_queue_wait_seconds", c.resource, "").GetHistogram()
			require.Equal(t, uint64(1), wait.GetSampleCount(), "dropped jobs must not count as started")
		})
	}
}

func TestRevalidationMetricsSeparateQueueWaitFromExecution(t *testing.T) {
	for _, fails := range []bool{false, true} {
		t.Run(fmt.Sprintf("fails=%t", fails), func(t *testing.T) {
			c, clk := newPausedRefreshCache(t)
			c.SetMany(t.Context(), map[string]string{"a": "old-a", "b": "old-b"})
			clk.Tick(2 * time.Minute)
			_, _, err := c.SWRMany(t.Context(), []string{"a", "b"}, func(_ context.Context, keys []string) (map[string]string, error) {
				clk.Tick(125 * time.Millisecond)
				if fails {
					return nil, fmt.Errorf("origin unavailable")
				}
				return map[string]string{keys[0]: "new-a", keys[1]: "new-b"}, nil
			}, func(err error) Op {
				if err != nil {
					return Noop
				}
				return WriteValue
			})
			require.NoError(t, err)
			for range 4 {
				_, _, err := readStale(c, "SWR", "a")
				require.NoError(t, err)
			}
			clk.Tick(3 * time.Second)
			(<-c.revalidateC)()

			queued := gatherCacheMetric(t, "unkey_cache_revalidation_enqueues_total", c.resource, "enqueued")
			require.Equal(t, float64(1), queued.GetCounter().GetValue(), "a batch occupies one queue slot, not one per key")
			deduplicated := gatherCacheMetric(t, "unkey_cache_revalidation_enqueues_total", c.resource, "deduplicated")
			require.Equal(t, float64(4), deduplicated.GetCounter().GetValue())
			depth := gatherCacheMetric(t, "unkey_cache_revalidation_queue_depth", c.resource, "").GetHistogram()
			require.Equal(t, uint64(1), depth.GetSampleCount(), "deduplicated attempts must not skew the queue depth distribution")
			require.Zero(t, depth.GetSampleSum())
			wait := gatherCacheMetric(t, "unkey_cache_revalidation_queue_wait_seconds", c.resource, "").GetHistogram()
			require.Equal(t, uint64(1), wait.GetSampleCount())
			require.Equal(t, float64(3), wait.GetSampleSum())
			duration := gatherCacheMetric(t, "unkey_cache_revalidation_duration_seconds", c.resource, "").GetHistogram()
			require.Equal(t, uint64(1), duration.GetSampleCount())
			require.Equal(t, 0.125, duration.GetSampleSum(), "execution time must exclude queue wait and include failed refreshes")
		})
	}
}

func TestStaleReadsDeduplicateQueuedRefreshes(t *testing.T) {
	c, clk := newPausedRefreshCache(t)
	c.SetMany(t.Context(), map[string]string{"a": "old-a", "b": "old-b"})
	clk.Tick(2 * time.Minute)

	for range 20 {
		for _, method := range []string{"SWR", "SWRMany", "SWRWithFallback"} {
			value, hit, err := readStale(c, method, "a")
			require.NoError(t, err)
			require.Equal(t, Hit, hit)
			require.Equal(t, "old-a", value)
		}
	}
	require.Len(t, c.revalidateC, 1, "all stale read methods must share queued-key deduplication")

	var refreshedKeys []string
	values, hits, err := c.SWRMany(t.Context(), []string{"a", "b"}, func(_ context.Context, keys []string) (map[string]string, error) {
		refreshedKeys = keys
		return map[string]string{"b": "new-b"}, nil
	}, func(error) Op { return WriteValue })
	require.NoError(t, err)
	require.Equal(t, map[string]string{"a": "old-a", "b": "old-b"}, values)
	require.Equal(t, map[string]CacheHit{"a": Hit, "b": Hit}, hits)
	require.Len(t, c.revalidateC, 2)
	(<-c.revalidateC)()
	(<-c.revalidateC)()
	require.Equal(t, []string{"b"}, refreshedKeys, "a batch must exclude keys already queued by a single read")

	clk.Tick(2 * time.Minute)
	_, _, err = readStale(c, "SWR", "a")
	require.NoError(t, err)
	require.Len(t, c.revalidateC, 1, "a completed refresh must not suppress the next stale refresh")
}

func TestConcurrentStaleReadsDeduplicateRunningRefresh(t *testing.T) {
	c, clk := newPausedRefreshCache(t)
	c.Set(t.Context(), "key", "stale")
	clk.Tick(2 * time.Minute)
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	releaseRefresh := sync.OnceFunc(func() { close(release) })
	_, _, err := c.SWR(t.Context(), "key", func(context.Context) (string, error) {
		close(started)
		<-release
		return "updated", nil
	}, func(error) Op { return WriteValue })
	require.NoError(t, err)
	refresh := <-c.revalidateC
	go func() {
		refresh()
		close(finished)
	}()
	t.Cleanup(func() {
		releaseRefresh()
		<-finished
	})
	<-started

	type result struct {
		value string
		hit   CacheHit
		err   error
	}
	results := make(chan result, 30)
	for range 10 {
		for _, method := range []string{"SWR", "SWRMany", "SWRWithFallback"} {
			go func() {
				value, hit, err := readStale(c, method, "key")
				results <- result{value, hit, err}
			}()
		}
	}
	for range 30 {
		got := <-results
		require.NoError(t, got.err)
		require.Equal(t, Hit, got.hit)
		require.Equal(t, "stale", got.value)
	}
	require.Empty(t, c.revalidateC, "running refreshes must not accumulate duplicate queued work")
	releaseRefresh()
	<-finished
	value, hit := c.Get(t.Context(), "key")
	require.Equal(t, Hit, hit)
	require.Equal(t, "updated", value)
}

func gatherCacheMetric(t *testing.T, name, resource, outcome string) *dto.Metric {
	t.Helper()
	families, err := testMetrics.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if labels["resource"] == resource && labels["outcome"] == outcome {
				return metric
			}
		}
	}
	t.Fatalf("missing metric %s for resource=%q outcome=%q", name, resource, outcome)
	return nil
}

func newPausedRefreshCache(t *testing.T) (*cache[string, string], *clock.TestClock) {
	t.Helper()
	clk := clock.NewTestClock()
	created, err := New(Config[string, string]{
		Fresh: time.Minute, Stale: time.Hour, MaxSize: 100, Resource: t.Name(), Clock: clk,
	})
	require.NoError(t, err)
	c := created.(*cache[string, string])
	started := make(chan struct{}, 10)
	finished := make(chan struct{}, 10)
	release := make(chan struct{})
	t.Cleanup(func() {
		c.Close()
		close(release)
		for range 10 {
			<-finished
		}
	})
	for range 10 {
		c.revalidateC <- func() {
			started <- struct{}{}
			<-release
			finished <- struct{}{}
		}
	}
	for range 10 {
		<-started
	}
	return c, clk
}

func readStale(c Cache[string, string], method, key string) (string, CacheHit, error) {
	ctx := context.Background()
	op := func(error) Op { return WriteValue }
	switch method {
	case "SWR":
		return c.SWR(ctx, key, func(context.Context) (string, error) {
			return "refreshed-" + key, nil
		}, op)
	case "SWRMany":
		values, hits, err := c.SWRMany(ctx, []string{key}, func(_ context.Context, keys []string) (map[string]string, error) {
			values := make(map[string]string, len(keys))
			for _, key := range keys {
				values[key] = "refreshed-" + key
			}
			return values, nil
		}, op)
		return values[key], hits[key], err
	case "SWRWithFallback":
		return c.SWRWithFallback(ctx, []string{"missing", key}, func(context.Context) (string, string, error) {
			return "refreshed-" + key, key, nil
		}, op)
	default:
		return "", Miss, fmt.Errorf("unknown stale read method: %s", method)
	}
}
