package cache_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/zen"
)

func TestWriteRead(t *testing.T) {

	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
	value, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "value", value)
}

func TestClose(t *testing.T) {

	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	c.Set(context.Background(), "key", "value")

	c.Close()
	// Close must be idempotent.
	c.Close()

	// Cached data stays readable after Close.
	value, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Hit, hit)
	require.Equal(t, "value", value)
}

func TestEviction(t *testing.T) {

	clk := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		MaxSize: 10_000,

		Fresh:    time.Second,
		Stale:    time.Second,
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
	clk.Tick(2 * time.Second)
	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Miss, hit)
}

func TestRefresh(t *testing.T) {

	clk := clock.NewTestClock()

	// count how many times we refreshed from origin
	refreshedFromOrigin := atomic.Int32{}

	c, err := cache.New(cache.Config[string, string]{
		MaxSize: 10_000,

		Fresh:    time.Second * 2,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)
	c.Set(context.Background(), "key", "value")
	clk.Tick(time.Second)

	for i := 0; i < 10; i++ {
		_, hit := c.Get(context.Background(), "key")
		require.Equal(t, cache.Hit, hit)
		clk.Tick(time.Second)
	}
	require.LessOrEqual(t, refreshedFromOrigin.Load(), int32(5))

}

func TestNull(t *testing.T) {

	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Second * 1,
		Stale:    time.Minute * 5,
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	c.SetNull(context.Background(), "key")

	_, hit := c.Get(context.Background(), "key")
	require.Equal(t, cache.Null, hit)

}

func TestNewRejectsInvalidConfig(t *testing.T) {
	_, err := cache.New(cache.Config[string, string]{
		MaxSize:  10,
		Resource: "test",
	})
	require.ErrorContains(t, err, "clock is required")

	for _, maxSize := range []int{0, -1} {
		_, err := cache.New(cache.Config[string, string]{
			MaxSize:  maxSize,
			Resource: "test",
			Clock:    clock.NewTestClock(),
		})
		require.ErrorContains(t, err, "MaxSize must be positive")
	}
}

func TestName(t *testing.T) {
	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  1,
		Resource: "keys",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	require.Equal(t, "keys", c.Name())
}

// TestDumpRestore moves entries between caches that share a clock. Restore
// keeps fresh and stale entries, gives them a new lifetime from the restoring
// cache's clock, and drops entries that are already expired.
func TestDumpRestore(t *testing.T) {
	clk := clock.NewTestClock()
	config := cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  10,
		Resource: "test",
		Clock:    clk,
	}
	source, err := cache.New(config)
	require.NoError(t, err)
	t.Cleanup(source.Close)
	source.Set(t.Context(), "expired", "x")
	clk.Tick(59 * time.Minute)
	source.Set(t.Context(), "stale", "s")
	clk.Tick(30 * time.Second)
	source.Set(t.Context(), "fresh", "f")
	clk.Tick(2 * time.Minute)

	b, err := source.Dump(t.Context())
	require.NoError(t, err)
	target, err := cache.New(config)
	require.NoError(t, err)
	t.Cleanup(target.Close)
	require.NoError(t, target.Restore(t.Context(), b))

	values, hits := target.GetMany(t.Context(), []string{"expired", "stale", "fresh"})
	require.Equal(t, map[string]string{
		"stale": "s",
		"fresh": "f",
	}, values)
	require.Equal(t, map[string]cache.CacheHit{
		"expired": cache.Miss,
		"stale":   cache.Hit,
		"fresh":   cache.Hit,
	}, hits)

	clk.Tick(59 * time.Minute)
	_, hit := target.Get(t.Context(), "stale")
	require.Equal(t, cache.Hit, hit, "restored entries start a new lifetime")

	require.ErrorContains(t, target.Restore(t.Context(), []byte("not json")), "failed to unmarshal cache data")
}

func TestDumpReportsUnencodableValues(t *testing.T) {
	c, err := cache.New(cache.Config[string, chan int]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  1,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	c.Set(t.Context(), "key", make(chan int))
	_, err = c.Dump(t.Context())
	require.ErrorContains(t, err, "failed to marshal cache data")
}

// An op that returns a value outside the Op constants writes nothing, and the
// read reports a miss while still returning what the origin gave it.
func TestSWRWithUnknownOpWritesNothing(t *testing.T) {
	c, err := cache.New(cache.Config[string, string]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  1,
		Resource: "test",
		Clock:    clock.NewTestClock(),
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	value, hit, err := c.SWR(t.Context(), "key", func(context.Context) (string, error) {
		return "origin", nil
	}, func(error) cache.Op { return cache.Op(99) })
	require.NoError(t, err)
	require.Equal(t, "origin", value)
	require.Equal(t, cache.Miss, hit)
	_, hit = c.Get(t.Context(), "key")
	require.Equal(t, cache.Miss, hit)
}

func TestNoopCacheStoresNothing(t *testing.T) {
	ctx := t.Context()
	c := cache.NewNoopCache[string, string]()
	t.Cleanup(c.Close)
	c.Set(ctx, "a", "1")
	c.SetMany(ctx, map[string]string{"b": "2"})
	c.SetNull(ctx, "c")
	c.SetNullMany(ctx, []string{"d"})
	c.Remove(ctx, "a")
	c.Clear(ctx)
	require.Equal(t, "noop", c.Name())

	_, hit := c.Get(ctx, "a")
	require.Equal(t, cache.Miss, hit)
	values, hits := c.GetMany(ctx, []string{"a", "b"})
	require.Empty(t, values)
	require.Equal(t, map[string]cache.CacheHit{
		"a": cache.Miss,
		"b": cache.Miss,
	}, hits)

	b, err := c.Dump(ctx)
	require.NoError(t, err)
	require.Empty(t, b)
	require.NoError(t, c.Restore(ctx, []byte("ignored")))

	originCalled := func(context.Context) (string, error) { t.Fatal("noop cache must not call the origin"); return "", nil }
	value, hit, err := c.SWR(ctx, "a", originCalled, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)
	require.Empty(t, value)
	require.Equal(t, cache.Miss, hit)

	values, hits, err = c.SWRMany(ctx, []string{"a"}, func(context.Context, []string) (map[string]string, error) {
		t.Fatal("noop cache must not call the origin")
		return nil, nil
	}, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)
	require.Empty(t, values)
	require.Equal(t, map[string]cache.CacheHit{"a": cache.Miss}, hits)

	value, hit, err = c.SWRWithFallback(ctx, []string{"a"}, func(context.Context) (string, string, error) {
		t.Fatal("noop cache must not call the origin")
		return "", "", nil
	}, func(error) cache.Op { return cache.WriteValue })
	require.NoError(t, err)
	require.Empty(t, value)
	require.Equal(t, cache.Miss, hit)
}

func TestScopedKeyRoundTrip(t *testing.T) {
	key := cache.ScopedKey{
		WorkspaceID: "ws_1",
		Key:         "name:with:colons",
	}
	require.Equal(t, "ws_1:name:with:colons", cache.ScopedKeyToString(key))
	parsed, err := cache.ScopedKeyFromString(key.String())
	require.NoError(t, err)
	require.Equal(t, key, parsed, "only the first colon separates the workspace")

	_, err = cache.ParseScopedKey("no-colon")
	require.ErrorContains(t, err, "invalid scoped key format")
}

// newHeldCache returns a cache whose background refreshes wait in the
// returned queue until the test runs them.
func newHeldCache(t *testing.T) (cache.Cache[string, string], *clock.TestClock, *refreshQueue) {
	t.Helper()
	clk := clock.NewTestClock()
	queue := &refreshQueue{
		refreshes: nil,
		limit:     100,
		sweep:     nil,
	}
	c, err := cache.New(cache.Config[string, string]{
		Fresh:     time.Minute,
		Stale:     time.Hour,
		MaxSize:   10,
		Resource:  "test",
		Clock:     clk,
		Scheduler: queue,
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c, clk, queue
}

func writeValue(error) cache.Op { return cache.WriteValue }

// TestTimingsUseTheClock records one timing per read and checks its name,
// status, and duration. Durations come from Config.Clock, so the test clock
// moving inside the origin is the only time that passes.
func TestTimingsUseTheClock(t *testing.T) {
	c, clk, _ := newHeldCache(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Unkey-Debug", "1")
	recorder := httptest.NewRecorder()
	session := &zen.Session{}
	require.NoError(t, session.Init(recorder, req, 0))
	ctx := zen.WithSession(context.Background(), session)
	origin := func(d time.Duration, value string) func(context.Context) (string, error) {
		return func(context.Context) (string, error) {
			clk.Tick(d)
			return value, nil
		}
	}
	fallbackOrigin := func(context.Context) (string, string, error) {
		clk.Tick(7 * time.Millisecond)
		return "w", "k", nil
	}

	c.Get(ctx, "k")
	_, _, err := c.SWR(ctx, "k", origin(5*time.Millisecond, "v"), writeValue)
	require.NoError(t, err)
	c.Get(ctx, "k")
	_, _, err = c.SWR(ctx, "k", origin(0, "unused"), writeValue)
	require.NoError(t, err)
	_, _, err = c.SWRWithFallback(ctx, []string{"k"}, fallbackOrigin, writeValue)
	require.NoError(t, err)
	clk.Tick(2 * time.Minute)
	c.Get(ctx, "k")
	_, _, err = c.SWR(ctx, "k", origin(0, "unused"), writeValue)
	require.NoError(t, err)
	_, _, err = c.SWRWithFallback(ctx, []string{"k"}, fallbackOrigin, writeValue)
	require.NoError(t, err)
	clk.Tick(2 * time.Hour)
	c.Get(ctx, "k")
	_, _, err = c.SWRWithFallback(ctx, []string{"k"}, fallbackOrigin, writeValue)
	require.NoError(t, err)

	require.Equal(t, []string{strings.Join([]string{
		"cache_get{cache=test,status=miss}=0ns",
		"cache_swr{cache=test,status=miss}=5ms",
		"cache_get{cache=test,status=fresh}=0ns",
		"cache_swr{cache=test,status=fresh}=0ns",
		"cache_swr_fallback{cache=test,status=fresh}=0ns",
		"cache_get{cache=test,status=stale}=0ns",
		"cache_swr{cache=test,status=stale}=0ns",
		"cache_swr_fallback{cache=test,status=stale}=0ns",
		"cache_get{cache=test,status=miss}=0ns",
		"cache_swr_fallback{cache=test,status=miss}=7ms",
	}, ",")}, recorder.Header().Values(timing.HeaderName))
}

// SWRMany treats a cached Null as settled and does not refresh it when it
// turns stale, unlike SWR.
func TestSWRManyLeavesStaleNullsAlone(t *testing.T) {
	c, clk, queue := newHeldCache(t)
	c.SetNull(t.Context(), "n")
	clk.Tick(2 * time.Minute)
	_, hits, err := c.SWRMany(t.Context(), []string{"n"}, func(context.Context, []string) (map[string]string, error) {
		t.Fatal("a cached Null must not reach the origin")
		return nil, nil
	}, writeValue)
	require.NoError(t, err)
	require.Equal(t, map[string]cache.CacheHit{"n": cache.Null}, hits)
	require.Empty(t, queue.refreshes)
}

// When op says WriteValue but the origin returned no map, SWRMany and its
// background refresh write nothing, not even Null for the keys they asked for.
func TestSWRManyWritesNothingWithoutValues(t *testing.T) {
	c, clk, queue := newHeldCache(t)
	noValues := func(context.Context, []string) (map[string]string, error) { return nil, nil }
	values, hits, err := c.SWRMany(t.Context(), []string{"a", "b"}, noValues, writeValue)
	require.NoError(t, err)
	require.Empty(t, values)
	require.Equal(t, map[string]cache.CacheHit{
		"a": cache.Miss,
		"b": cache.Miss,
	}, hits)
	_, hit := c.Get(t.Context(), "a")
	require.Equal(t, cache.Miss, hit)

	c.Set(t.Context(), "c", "vc")
	clk.Tick(2 * time.Minute)
	_, _, err = c.SWRMany(t.Context(), []string{"c"}, noValues, writeValue)
	require.NoError(t, err)
	require.Len(t, queue.refreshes, 1)
	queue.refreshes[0]()
	value, hit := c.Get(t.Context(), "c")
	require.Equal(t, "vc", value, "a refresh without values must keep the stale entry")
	require.Equal(t, cache.Hit, hit)
}

// SWRWithFallback skips a missing candidate and deletes an expired one before
// it serves a later fresh candidate.
func TestSWRWithFallbackServesALaterCandidate(t *testing.T) {
	c, clk, queue := newHeldCache(t)
	c.Set(t.Context(), "b", "vb")
	clk.Tick(2 * time.Hour)
	c.Set(t.Context(), "c", "vc")
	value, hit, err := c.SWRWithFallback(t.Context(), []string{"a", "b", "c"}, func(context.Context) (string, string, error) {
		t.Fatal("a cached candidate must not reach the origin")
		return "", "", nil
	}, writeValue)
	require.NoError(t, err)
	require.Equal(t, "vc", value)
	require.Equal(t, cache.Hit, hit)
	require.Empty(t, queue.refreshes, "a fresh candidate must not schedule a refresh")
}

func TestSWRWithFallbackWritesNullWithoutError(t *testing.T) {
	c, _, _ := newHeldCache(t)
	value, hit, err := c.SWRWithFallback(t.Context(), []string{"a"}, func(context.Context) (string, string, error) {
		return "", "canonical", nil
	}, func(error) cache.Op { return cache.WriteNull })
	require.NoError(t, err)
	require.Empty(t, value)
	require.Equal(t, cache.Null, hit)
	_, hit = c.Get(t.Context(), "canonical")
	require.Equal(t, cache.Null, hit)
}

// A background fallback refresh that fails keeps the stale entry, even when
// op would write a value for that error.
func TestSWRWithFallbackRefreshKeepsEntryOnError(t *testing.T) {
	c, clk, queue := newHeldCache(t)
	c.Set(t.Context(), "k", "old")
	clk.Tick(2 * time.Minute)
	value, hit, err := c.SWRWithFallback(t.Context(), []string{"k"}, func(context.Context) (string, string, error) {
		return "new", "k", errors.New("origin unavailable")
	}, writeValue)
	require.NoError(t, err)
	require.Equal(t, "old", value)
	require.Equal(t, cache.Hit, hit)
	require.Len(t, queue.refreshes, 1)
	queue.refreshes[0]()
	value, hit = c.Get(t.Context(), "k")
	require.Equal(t, "old", value)
	require.Equal(t, cache.Hit, hit)
}

// With the default scheduler, a ticker on Config.Clock deletes expired
// entries that nobody reads and keeps the ones that are still alive.
func TestDefaultSchedulerSweepsExpiredEntries(t *testing.T) {
	clk := clock.NewTestClock()
	c, err := cache.New(cache.Config[string, string]{
		Fresh:     time.Second,
		Stale:     2 * time.Second,
		MaxSize:   10,
		Resource:  "test",
		Clock:     clk,
		Scheduler: nil,
	})
	require.NoError(t, err)
	t.Cleanup(c.Close)

	c.Set(t.Context(), "old", "v")
	clk.Tick(59 * time.Second)
	c.Set(t.Context(), "new", "v")
	clk.Tick(time.Second)

	require.Eventually(t, func() bool {
		b, err := c.Dump(t.Context())
		require.NoError(t, err)
		return !strings.Contains(string(b), `"old"`)
	}, 5*time.Second, time.Millisecond)
	_, hit := c.Get(t.Context(), "new")
	require.Equal(t, cache.Hit, hit)
}
