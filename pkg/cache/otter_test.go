package cache

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestCacheConfigurationLimits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		size  int
		stale time.Duration
	}{
		{"zero size", 0, time.Minute},
		{"negative size", -1, time.Minute},
		{"zero TTL", 1, 0},
		{"negative TTL", 1, -time.Minute},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(Config[string, string]{
				Fresh:    time.Minute,
				Stale:    tc.stale,
				MaxSize:  tc.size,
				Resource: t.Name(),
				Clock:    clock.New(),
			})
			require.Error(t, err)
			require.Nil(t, c)
		})
	}
}

func TestOverwriteResetsExpiration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, err := New(Config[string, string]{
			Fresh:    time.Minute,
			Stale:    2 * time.Minute,
			MaxSize:  10,
			Resource: t.Name(),
			Clock:    clock.New(),
		})
		require.NoError(t, err)
		t.Cleanup(c.Close)
		t.Cleanup(func() { c.(*cache[string, string]).otter.StopAllGoroutines() })
		ctx := context.Background()
		c.Set(ctx, "key", "first")
		time.Sleep(90 * time.Second)
		c.Set(ctx, "key", "replacement")
		time.Sleep(45 * time.Second)
		value, hit := c.Get(ctx, "key")
		require.Equal(t, Hit, hit)
		require.Equal(t, "replacement", value)
		time.Sleep(80 * time.Second)
		_, hit = c.Get(ctx, "key")
		require.Equal(t, Miss, hit)
	})
}
