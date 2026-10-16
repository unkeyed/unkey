package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
	"github.com/unkeyed/unkey/pkg/uid"
)

var testMetrics = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(testMetrics)
	os.Exit(m.Run())
}

func TestStaleReadsDoNotWaitForRefreshQueue(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"SWR", "SWRMany", "SWRWithFallback"} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
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
				receive(t, returned)
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
			queuedRefresh(t, c)()
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
	t.Parallel()
	for _, fails := range []bool{false, true} {
		t.Run(fmt.Sprintf("fails=%t", fails), func(t *testing.T) {
			t.Parallel()
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
			c.Set(t.Context(), "fresh", "fresh")
			_, _, err = c.SWRMany(t.Context(), []string{"fresh"}, func(context.Context, []string) (map[string]string, error) {
				t.Error("a fresh read must not read the origin")
				return nil, nil
			}, func(error) Op { return WriteValue })
			require.NoError(t, err)
			for range 4 {
				_, _, err := readStale(c, "SWR", "a")
				require.NoError(t, err)
			}
			clk.Tick(3 * time.Second)
			queuedRefresh(t, c)()

			queued := gatherCacheMetric(t, "unkey_cache_revalidation_enqueues_total", c.resource, "enqueued")
			require.Equal(t, float64(1), queued.GetCounter().GetValue(), "a batch occupies one queue slot, not one per key")
			deduplicated := gatherCacheMetric(t, "unkey_cache_revalidation_enqueues_total", c.resource, "deduplicated")
			require.Equal(t, float64(4), deduplicated.GetCounter().GetValue(), "a read without stale keys must not count as an enqueue attempt")
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
	t.Parallel()
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
	queuedRefresh(t, c)()
	queuedRefresh(t, c)()
	require.Equal(t, []string{"b"}, refreshedKeys, "a batch must exclude keys already queued by a single read")

	clk.Tick(2 * time.Minute)
	_, _, err = readStale(c, "SWR", "a")
	require.NoError(t, err)
	require.Len(t, c.revalidateC, 1, "a completed refresh must not suppress the next stale refresh")
}

func TestConcurrentStaleReadsDeduplicateRunningRefresh(t *testing.T) {
	t.Parallel()
	c, clk := newPausedRefreshCache(t)
	c.Set(t.Context(), "key", "stale")
	clk.Tick(2 * time.Minute)
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	releaseRefresh := sync.OnceFunc(func() { close(release) })
	_, _, err := c.SWR(t.Context(), "key", func(context.Context) (string, error) {
		close(started)
		if err := waitForRelease(release); err != nil {
			return "", err
		}
		return "updated", nil
	}, func(error) Op { return WriteValue })
	require.NoError(t, err)
	refresh := queuedRefresh(t, c)
	go func() {
		refresh()
		close(finished)
	}()
	t.Cleanup(func() {
		releaseRefresh()
		receive(t, finished)
	})
	receive(t, started)

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
		got := receive(t, results)
		require.NoError(t, got.err)
		require.Equal(t, Hit, got.hit)
		require.Equal(t, "stale", got.value)
	}
	require.Empty(t, c.revalidateC, "running refreshes must not accumulate duplicate queued work")
	releaseRefresh()
	receive(t, finished)
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
		Fresh: time.Minute, Stale: time.Hour, MaxSize: 100, Resource: uid.New(uid.TestPrefix), Clock: clk,
	})
	require.NoError(t, err)
	c := created.(*cache[string, string])
	started := make(chan struct{}, 10)
	finished := make(chan struct{}, 10)
	release := make(chan struct{})
	paused := 0
	t.Cleanup(func() {
		c.Close()
		close(release)
		for range paused {
			receive(t, finished)
		}
	})
	for range 10 {
		select {
		case c.revalidateC <- func() {
			started <- struct{}{}
			<-release
			finished <- struct{}{}
		}:
		default:
			t.Fatal("the refresh queue has no room")
		}
	}
	for range 10 {
		receive(t, started)
		paused++
	}
	return c, clk
}

// queuedRefresh takes the refresh that a stale read queued. Stale reads queue
// before they return and newPausedRefreshCache keeps workers from taking it,
// so a missing refresh fails at once.
func queuedRefresh(t *testing.T, c *cache[string, string]) func() {
	t.Helper()
	select {
	case refresh := <-c.revalidateC:
		return refresh
	default:
		t.Fatal("no refresh is queued")
		return nil
	}
}

// waitForRelease bounds how long an origin blocks, so a cache that runs a
// background read on the caller's goroutine fails the test instead of hanging.
func waitForRelease(release <-chan struct{}) error {
	select {
	case <-release:
		return nil
	case <-time.After(5 * time.Second):
		return errors.New("the origin was never released")
	}
}

// receive fails the test instead of blocking forever when a broken cache never
// sends on ch.
func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting on a channel")
		var zero T
		return zero
	}
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

// TestBackgroundBatchRefreshWritesWhatOpDecides guarantees that a background
// batch refresh caches a key the origin no longer returns as null, so a deleted
// row stops being served once the refresh lands, and that a failed refresh
// keeps serving the stale values.
func TestBackgroundBatchRefreshWritesWhatOpDecides(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		origin     func(keys []string) (map[string]string, error)
		wantValues map[string]string
		wantHits   map[string]CacheHit
	}{
		{
			name: "omitted key becomes null",
			origin: func([]string) (map[string]string, error) {
				return map[string]string{"a": "new-a"}, nil
			},
			wantValues: map[string]string{"a": "new-a", "b": ""},
			wantHits:   map[string]CacheHit{"a": Hit, "b": Null},
		},
		{
			name: "not found writes null for every key",
			origin: func([]string) (map[string]string, error) {
				return nil, sql.ErrNoRows
			},
			wantValues: map[string]string{"a": "", "b": ""},
			wantHits:   map[string]CacheHit{"a": Null, "b": Null},
		},
		{
			name: "failure keeps stale values",
			origin: func([]string) (map[string]string, error) {
				return nil, errors.New("origin unavailable")
			},
			wantValues: map[string]string{"a": "old-a", "b": "old-b"},
			wantHits:   map[string]CacheHit{"a": Hit, "b": Hit},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c, clk := newPausedRefreshCache(t)
			c.SetMany(t.Context(), map[string]string{"a": "old-a", "b": "old-b"})
			clk.Tick(2 * time.Minute)

			_, _, err := c.SWRMany(t.Context(), []string{"a", "b"}, func(_ context.Context, keys []string) (map[string]string, error) {
				return tt.origin(keys)
			}, func(err error) Op {
				if db.IsNotFound(err) {
					return WriteNull
				}
				if err != nil {
					return Noop
				}
				return WriteValue
			})
			require.NoError(t, err)
			queuedRefresh(t, c)()

			values, hits := c.GetMany(t.Context(), []string{"a", "b"})
			require.Equal(t, tt.wantValues, values)
			require.Equal(t, tt.wantHits, hits)
		})
	}
}

// TestBatchRefreshReleasesEveryKey guarantees that every key of a batch
// refresh can be refreshed again once the batch has run or was dropped from a
// full queue, so one lost batch cannot stop a key from refreshing.
func TestBatchRefreshReleasesEveryKey(t *testing.T) {
	t.Parallel()
	c, clk := newPausedRefreshCache(t)
	c.SetMany(t.Context(), map[string]string{"a": "old-a", "b": "old-b"})
	clk.Tick(2 * time.Minute)
	var refreshed [][]string
	readStaleBatch := func() {
		t.Helper()
		_, _, err := c.SWRMany(t.Context(), []string{"a", "b"}, func(_ context.Context, keys []string) (map[string]string, error) {
			refreshed = append(refreshed, keys)
			return map[string]string{"a": "new-a", "b": "new-b"}, nil
		}, func(error) Op { return WriteValue })
		require.NoError(t, err)
	}

	for range cap(c.revalidateC) {
		c.revalidateC <- func() {}
	}
	readStaleBatch()
	for len(c.revalidateC) > 0 {
		<-c.revalidateC
	}

	readStaleBatch()
	queuedRefresh(t, c)()
	clk.Tick(2 * time.Minute)
	readStaleBatch()
	queuedRefresh(t, c)()

	require.Equal(t, [][]string{{"a", "b"}, {"a", "b"}}, refreshed)
}

// TestFailedBackgroundRefreshIsLogged guarantees that a background refresh
// that fails logs a warning with the error, because no caller sees that error,
// and that a not found result, which op handles, logs nothing.
func TestFailedBackgroundRefreshIsLogged(t *testing.T) {
	logs := loggertest.Install(t)
	c, clk := newPausedRefreshCache(t)
	originErr := errors.New("origin unavailable")
	tests := []struct {
		method  string
		message string
		attrs   func(key string) map[string]any
	}{
		{method: "SWR", message: "failed to revalidate", attrs: func(key string) map[string]any {
			return map[string]any{"error": originErr.Error(), "key": key}
		}},
		{method: "SWRMany", message: "failed to revalidate many", attrs: func(key string) map[string]any {
			return map[string]any{"error": originErr.Error(), "keys": []string{key}, "resource": c.resource}
		}},
		{method: "SWRWithFallback", message: "failed to revalidate with canonical key", attrs: func(string) map[string]any {
			return map[string]any{"error": originErr.Error()}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			for _, err := range []error{originErr, sql.ErrNoRows} {
				key := fmt.Sprintf("%s-%v", tt.method, err)
				c.Set(t.Context(), key, "stale")
				clk.Tick(2 * time.Minute)
				before := logs.Snapshot()

				value, hit, readErr := readStaleFailing(c, tt.method, key, err)
				require.NoError(t, readErr)
				require.Equal(t, Hit, hit)
				require.Equal(t, "stale", value)
				queuedRefresh(t, c)()

				records := logs.Since(before)
				if err == sql.ErrNoRows {
					require.Empty(t, records, "a not found refresh must not log")
					continue
				}
				require.Len(t, records, 1)
				require.Equal(t, slog.LevelWarn, records[0].Level)
				require.Equal(t, tt.message, records[0].Message)
				require.Equal(t, tt.attrs(key), loggertest.FlatAttrs(records[0]))
			}
		})
	}
}

func readStaleFailing(c Cache[string, string], method, key string, originErr error) (string, CacheHit, error) {
	ctx := context.Background()
	op := func(err error) Op {
		if db.IsNotFound(err) {
			return WriteNull
		}
		return Noop
	}
	switch method {
	case "SWR":
		return c.SWR(ctx, key, func(context.Context) (string, error) {
			return "", originErr
		}, op)
	case "SWRMany":
		values, hits, err := c.SWRMany(ctx, []string{key}, func(context.Context, []string) (map[string]string, error) {
			return nil, originErr
		}, op)
		return values[key], hits[key], err
	case "SWRWithFallback":
		return c.SWRWithFallback(ctx, []string{key}, func(context.Context) (string, string, error) {
			return "", "", originErr
		}, op)
	default:
		return "", Miss, fmt.Errorf("unknown stale read method: %s", method)
	}
}
