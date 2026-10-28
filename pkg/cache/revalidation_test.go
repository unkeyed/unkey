package cache

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

var testMetrics = prometheus.DefaultGatherer

func TestStaleReadsDoNotWaitForRefreshQueue(t *testing.T) {
	for _, method := range []string{"SWR", "SWRMany", "SWRWithFallback"} {
		t.Run(method, func(t *testing.T) {
			c, clk := newPausedRefreshCache(t)
			c.Set(t.Context(), "target", "stale")
			clk.Tick(2 * time.Minute)
			for range cap(queued(c)) {
				queued(c) <- func() {}
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

			for len(queued(c)) > 0 {
				<-queued(c)
			}
			_, _, err := readStale(c, method, "target")
			require.NoError(t, err)
			require.Len(t, queued(c), 1, "a dropped refresh must be retryable")
			(<-queued(c))()
			value, hit := c.Get(t.Context(), "target")
			require.Equal(t, Hit, hit)
			require.Equal(t, "refreshed-target", value)

			c.Close()
			clk.Tick(2 * time.Minute)
			value, hit, err = readStale(c, method, "target")
			require.NoError(t, err)
			require.Equal(t, Hit, hit)
			require.Equal(t, "refreshed-target", value)
			require.Empty(t, queued(c))
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
			c.SetMany(t.Context(), map[string]string{
				"a": "old-a",
				"b": "old-b",
			})
			clk.Tick(2 * time.Minute)
			_, _, err := c.SWRMany(t.Context(), []string{"a", "b"}, func(_ context.Context, keys []string) (map[string]string, error) {
				clk.Tick(125 * time.Millisecond)
				if fails {
					return nil, fmt.Errorf("origin unavailable")
				}
				return map[string]string{
					keys[0]: "new-a",
					keys[1]: "new-b",
				}, nil
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
			(<-queued(c))()

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
	c.SetMany(t.Context(), map[string]string{
		"a": "old-a",
		"b": "old-b",
	})
	clk.Tick(2 * time.Minute)

	for range 20 {
		for _, method := range []string{"SWR", "SWRMany", "SWRWithFallback"} {
			value, hit, err := readStale(c, method, "a")
			require.NoError(t, err)
			require.Equal(t, Hit, hit)
			require.Equal(t, "old-a", value)
		}
	}
	require.Len(t, queued(c), 1, "all stale read methods must share queued-key deduplication")

	var refreshedKeys []string
	values, hits, err := c.SWRMany(t.Context(), []string{"a", "b"}, func(_ context.Context, keys []string) (map[string]string, error) {
		refreshedKeys = keys
		return map[string]string{"b": "new-b"}, nil
	}, func(error) Op { return WriteValue })
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"a": "old-a",
		"b": "old-b",
	}, values)
	require.Equal(t, map[string]CacheHit{
		"a": Hit,
		"b": Hit,
	}, hits)
	require.Len(t, queued(c), 2)
	(<-queued(c))()
	(<-queued(c))()
	require.Equal(t, []string{"b"}, refreshedKeys, "a batch must exclude keys already queued by a single read")

	clk.Tick(2 * time.Minute)
	_, _, err = readStale(c, "SWR", "a")
	require.NoError(t, err)
	require.Len(t, queued(c), 1, "a completed refresh must not suppress the next stale refresh")
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
	refresh := <-queued(c)
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
	require.Empty(t, queued(c), "running refreshes must not accumulate duplicate queued work")
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
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  100,
		Resource: t.Name(),
		Clock:    clk,
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
		queued(c) <- func() {
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

// queued returns the refresh queue of c's default worker pool.
func queued(c *cache[string, string]) chan func() {
	return c.scheduler.(*workers).refreshes
}

// counterValue returns the value of the counter with the given labels, or 0
// when nothing has incremented it yet.
func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := testMetrics.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			got := map[string]string{}
			for _, label := range metric.GetLabel() {
				got[label.GetName()] = label.GetValue()
			}
			matches := true
			for k, v := range labels {
				if got[k] != v {
					matches = false
				}
			}
			if matches {
				return metric.GetCounter().GetValue()
			}
		}
	}
	return 0
}

func TestDeletedMetricCountsEachReason(t *testing.T) {
	clk := clock.NewTestClock()
	c, err := New(Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  2,
		Resource: t.Name(),
		Clock:    clk,
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	ctx := t.Context()
	deleted := func(reason string) float64 {
		return counterValue(t, "unkey_cache_deleted_total", map[string]string{
			"resource": t.Name(),
			"reason":   reason,
		})
	}

	c.Set(ctx, "a", "1")
	c.Set(ctx, "a", "2")
	c.Set(ctx, "b", "1")
	c.Set(ctx, "c", "1")
	c.Remove(ctx, "b", "c", "missing")
	for _, key := range []string{"a", "b", "c"} {
		_, hit := c.Get(ctx, key)
		require.Equal(t, Miss, hit, key)
	}
	c.Set(ctx, "d", "1")
	clk.Tick(2 * time.Hour)
	c.Get(ctx, "d")
	c.Get(ctx, "d")

	require.Equal(t, float64(1), deleted("Replaced"))
	require.Equal(t, float64(1), deleted("Size"))
	require.Equal(t, float64(2), deleted("Explicit"), "Remove must delete every key it is given")
	require.Equal(t, float64(1), deleted("Expired"), "a read of an already deleted key must not count again")

	op := func(error) Op { return WriteValue }
	origin := func(_ context.Context, keys []string) (map[string]string, error) {
		values := map[string]string{}
		for _, key := range keys {
			values[key] = "v"
		}
		return values, nil
	}
	for range 2 {
		_, _, err := c.SWRMany(ctx, []string{"e", "f"}, origin, op)
		require.NoError(t, err)
	}
	require.Equal(t, float64(1), deleted("Replaced"), "SWRMany must write each fetched key once")
	require.Zero(t, counterValue(t, "unkey_cache_revalidation_enqueues_total", map[string]string{
		"resource": t.Name(),
		"outcome":  "deduplicated",
	}),
		"fresh reads must not ask for a refresh")
}

// A refresh for several keys must release all of them, whether it runs or is
// dropped, or the keys it missed would never be refreshed again.
func TestBatchRefreshReleasesEveryKey(t *testing.T) {
	c, clk := newPausedRefreshCache(t)
	ctx := t.Context()
	var refreshed []string
	origin := func(_ context.Context, keys []string) (map[string]string, error) {
		refreshed = keys
		return map[string]string{
			"a": "new-a",
			"b": "new-b",
		}, nil
	}
	op := func(error) Op { return WriteValue }
	c.SetMany(ctx, map[string]string{
		"a": "old-a",
		"b": "old-b",
	})

	clk.Tick(2 * time.Minute)
	_, _, err := c.SWRMany(ctx, []string{"a", "b"}, origin, op)
	require.NoError(t, err)
	(<-queued(c))()
	require.ElementsMatch(t, []string{"a", "b"}, refreshed)

	clk.Tick(2 * time.Minute)
	for range cap(queued(c)) - len(queued(c)) {
		queued(c) <- func() {}
	}
	_, _, err = c.SWRMany(ctx, []string{"a", "b"}, origin, op)
	require.NoError(t, err)
	for len(queued(c)) > 0 {
		<-queued(c)
	}

	refreshed = nil
	_, _, err = c.SWRMany(ctx, []string{"a", "b"}, origin, op)
	require.NoError(t, err)
	require.Len(t, queued(c), 1, "a dropped batch must release every key")
	(<-queued(c))()
	require.ElementsMatch(t, []string{"a", "b"}, refreshed)
}

// A caller can stop waiting for a load after the load has finished and a newer
// load for the same key has started. Its leaveLoad must cancel only its own
// load and leave the newer one registered, or later callers would start a
// duplicate origin read.
func TestLeaveLoadKeepsANewerLoad(t *testing.T) {
	created, err := New(Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: t.Name(),
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	c := created.(*cache[string, string])
	t.Cleanup(c.Close)

	finished := &inflight[string]{
		done:    make(chan struct{}),
		cancel:  func() {},
		waiters: 1,
		value:   "",
		hit:     Miss,
		err:     nil,
	}
	newer := &inflight[string]{
		done:    make(chan struct{}),
		cancel:  func() {},
		waiters: 1,
		value:   "",
		hit:     Miss,
		err:     nil,
	}
	c.loads["key"] = newer

	c.leaveLoad("key", finished)

	require.Same(t, newer, c.loads["key"])
}

// Every read records one duration under its operation and lowercase status.
func TestOperationDurationCountsEachRead(t *testing.T) {
	clk := clock.NewTestClock()
	created, err := New(Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: t.Name(),
		Clock:    clk,
	})
	require.NoError(t, err)
	c := created.(*cache[string, string])
	t.Cleanup(c.Close)

	c.Get(t.Context(), "key")
	c.Set(t.Context(), "key", "value")
	c.Get(t.Context(), "key")
	_, _, err = c.SWR(t.Context(), "key", func(context.Context) (string, error) {
		return "value", nil
	}, func(error) Op { return WriteValue })
	require.NoError(t, err)
	clk.Tick(2 * time.Minute)
	c.Get(t.Context(), "key")

	counts := map[string]uint64{}
	families, err := testMetrics.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "unkey_cache_operation_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			labels := map[string]string{}
			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}
			if n := metric.GetHistogram().GetSampleCount(); labels["resource"] == t.Name() && n > 0 {
				counts[labels["operation"]+"/"+labels["status"]] = n
			}
		}
	}
	require.Equal(t, map[string]uint64{
		"get/miss":  1,
		"get/fresh": 1,
		"get/stale": 1,
		"swr/fresh": 1,
	}, counts)
}
