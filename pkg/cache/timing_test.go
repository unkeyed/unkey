package cache_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/timing"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

// TestReadsReportCacheStatusInDebugTiming guarantees that the debug timing
// entry of a read names the cache and says whether the read was served fresh,
// served stale, or missed. Operators use this entry to explain the latency of
// a single request.
func TestReadsReportCacheStatusInDebugTiming(t *testing.T) {
	op := func(error) cache.Op { return cache.WriteValue }
	reads := []struct {
		name  string
		entry string
		read  func(ctx context.Context, c cache.Cache[string, string])
	}{
		{name: "Get", entry: "cache_get", read: func(ctx context.Context, c cache.Cache[string, string]) {
			c.Get(ctx, "key")
		}},
		{name: "SWR", entry: "cache_swr", read: func(ctx context.Context, c cache.Cache[string, string]) {
			_, _, err := c.SWR(ctx, "key", func(context.Context) (string, error) { return "origin", nil }, op)
			require.NoError(t, err)
		}},
		{name: "SWRWithFallback", entry: "cache_swr_fallback", read: func(ctx context.Context, c cache.Cache[string, string]) {
			_, _, err := c.SWRWithFallback(ctx, []string{"key"}, func(context.Context) (string, string, error) {
				return "origin", "key", nil
			}, op)
			require.NoError(t, err)
		}},
	}
	states := []struct {
		name   string
		status string
		setup  func(c cache.Cache[string, string], clk *clock.TestClock)
	}{
		{name: "missing", status: "miss", setup: func(cache.Cache[string, string], *clock.TestClock) {}},
		{name: "fresh", status: "fresh", setup: func(c cache.Cache[string, string], _ *clock.TestClock) {
			c.Set(context.Background(), "key", "cached")
		}},
		{name: "fresh null", status: "fresh", setup: func(c cache.Cache[string, string], _ *clock.TestClock) {
			c.SetNull(context.Background(), "key")
		}},
		{name: "fresh null from batch", status: "fresh", setup: func(c cache.Cache[string, string], _ *clock.TestClock) {
			c.SetNullMany(context.Background(), []string{"key"})
		}},
		{name: "stale", status: "stale", setup: func(c cache.Cache[string, string], clk *clock.TestClock) {
			c.Set(context.Background(), "key", "cached")
			clk.Tick(2 * time.Minute)
		}},
		{name: "expired", status: "miss", setup: func(c cache.Cache[string, string], clk *clock.TestClock) {
			c.Set(context.Background(), "key", "cached")
			clk.Tick(2 * time.Hour)
		}},
	}
	for _, read := range reads {
		for _, state := range states {
			t.Run(read.name+"/"+state.name, func(t *testing.T) {
				clk := clock.NewTestClock()
				resource := uid.New(uid.TestPrefix)
				c, err := cache.New(cache.Config[string, string]{
					Fresh:    time.Minute,
					Stale:    time.Hour,
					MaxSize:  10,
					Resource: resource,
					Clock:    clk,
				})
				require.NoError(t, err)
				t.Cleanup(c.Close)
				state.setup(c, clk)

				ctx, header := debugTimingContext(t)
				read.read(ctx, c)

				values := header.Values(timing.HeaderName)
				require.Len(t, values, 1)
				entry, err := timing.ParseEntry(values[0])
				require.NoError(t, err)
				require.Equal(t, read.entry, entry.Name)
				require.Equal(t, map[string]string{"cache": resource, "status": state.status}, entry.Attributes)
			})
		}
	}
}

func debugTimingContext(t *testing.T) (context.Context, http.Header) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Unkey-Debug", "true")
	recorder := httptest.NewRecorder()
	session := &zen.Session{}
	require.NoError(t, session.Init(recorder, req, 0))
	return zen.WithSession(t.Context(), session), recorder.Header()
}
