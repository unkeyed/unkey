package cache_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

func TestGetMany(t *testing.T) {
	ctx := context.Background()
	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Logger:   logging.NewNoop(),
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	t.Run("all keys miss", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}
		values, hits := c.GetMany(ctx, keys)

		require.Len(t, values, 0)
		require.Len(t, hits, 3)
		for _, key := range keys {
			require.Equal(t, cache.Miss, hits[key])
		}
	})

	t.Run("some keys hit", func(t *testing.T) {
		// Set some values
		c.Set(ctx, "key1", "value1")
		c.Set(ctx, "key2", "value2")

		keys := []string{"key1", "key2", "key3", "key4"}
		values, hits := c.GetMany(ctx, keys)

		require.Len(t, values, 2)
		require.Len(t, hits, 4)

		require.Equal(t, "value1", values["key1"])
		require.Equal(t, cache.Hit, hits["key1"])

		require.Equal(t, "value2", values["key2"])
		require.Equal(t, cache.Hit, hits["key2"])

		require.Equal(t, cache.Miss, hits["key3"])
		require.Equal(t, cache.Miss, hits["key4"])
	})

	t.Run("all keys hit", func(t *testing.T) {
		// Set all values
		c.Set(ctx, "a", "va")
		c.Set(ctx, "b", "vb")
		c.Set(ctx, "c", "vc")

		keys := []string{"a", "b", "c"}
		values, hits := c.GetMany(ctx, keys)

		require.Len(t, values, 3)
		require.Len(t, hits, 3)

		for _, key := range keys {
			require.Equal(t, cache.Hit, hits[key])
			require.Equal(t, "v"+key, values[key])
		}
	})

	t.Run("null values", func(t *testing.T) {
		c.SetNull(ctx, "null1")
		c.SetNull(ctx, "null2")

		keys := []string{"null1", "null2"}
		values, hits := c.GetMany(ctx, keys)

		require.Len(t, values, 2)
		require.Len(t, hits, 2)

		require.Equal(t, cache.Null, hits["null1"])
		require.Equal(t, cache.Null, hits["null2"])
		require.Equal(t, "", values["null1"])
		require.Equal(t, "", values["null2"])
	})

	t.Run("empty keys slice", func(t *testing.T) {
		values, hits := c.GetMany(ctx, []string{})

		require.Len(t, values, 0)
		require.Len(t, hits, 0)
	})
}

func TestGetMany_Eviction(t *testing.T) {
	ctx := context.Background()
	clk := clock.NewTestClock()

	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Second,
		Stale:    time.Second,
		Logger:   logging.NewNoop(),
		Resource: "test",
		Clock:    clk,
	})
	require.NoError(t, err)

	// Set values
	c.Set(ctx, "key1", "value1")
	c.Set(ctx, "key2", "value2")

	// Move past stale time
	clk.Tick(2 * time.Second)

	keys := []string{"key1", "key2"}
	values, hits := c.GetMany(ctx, keys)

	require.Len(t, values, 0)
	require.Len(t, hits, 2)
	require.Equal(t, cache.Miss, hits["key1"])
	require.Equal(t, cache.Miss, hits["key2"])
}

func TestSetMany(t *testing.T) {
	ctx := context.Background()
	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Logger:   logging.NewNoop(),
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	t.Run("set multiple values", func(t *testing.T) {
		values := map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		c.SetMany(ctx, values)

		// Verify all values are set
		for key, expectedValue := range values {
			value, hit := c.Get(ctx, key)
			require.Equal(t, cache.Hit, hit)
			require.Equal(t, expectedValue, value)
		}
	})

	t.Run("overwrite existing values", func(t *testing.T) {
		c.Set(ctx, "old", "old_value")

		c.SetMany(ctx, map[string]string{
			"old": "new_value",
			"new": "new_value2",
		})

		value, hit := c.Get(ctx, "old")
		require.Equal(t, cache.Hit, hit)
		require.Equal(t, "new_value", value)

		value, hit = c.Get(ctx, "new")
		require.Equal(t, cache.Hit, hit)
		require.Equal(t, "new_value2", value)
	})

	t.Run("empty map", func(t *testing.T) {
		c.SetMany(ctx, map[string]string{})
		// Should not panic
	})
}

func TestSetNullMany(t *testing.T) {
	ctx := context.Background()
	c, err := cache.New(cache.Config[string, string]{
		MaxSize:  10_000,
		Fresh:    time.Minute,
		Stale:    time.Minute * 5,
		Logger:   logging.NewNoop(),
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)

	t.Run("set multiple null values", func(t *testing.T) {
		keys := []string{"null1", "null2", "null3"}
		c.SetNullMany(ctx, keys)

		// Verify all are null
		for _, key := range keys {
			value, hit := c.Get(ctx, key)
			require.Equal(t, cache.Null, hit)
			require.Equal(t, "", value)
		}
	})

	t.Run("overwrite existing values with null", func(t *testing.T) {
		c.Set(ctx, "existing", "value")

		c.SetNullMany(ctx, []string{"existing"})

		value, hit := c.Get(ctx, "existing")
		require.Equal(t, cache.Null, hit)
		require.Equal(t, "", value)
	})

	t.Run("empty slice", func(t *testing.T) {
		c.SetNullMany(ctx, []string{})
		// Should not panic
	})
}

func TestSWRMany(t *testing.T) {
	ctx := context.Background()
	mockClock := clock.NewTestClock()

	c, err := cache.New(cache.Config[string, string]{
		Fresh:    1 * time.Minute,
		Stale:    5 * time.Minute,
		Logger:   logging.NewNoop(),
		MaxSize:  100,
		Resource: "test",
		Clock:    mockClock,
	})
	require.NoError(t, err)

	t.Run("all keys miss - fetch from origin", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}
		values, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			result := make(map[string]string)
			for _, k := range keysToFetch {
				result[k] = "value_" + k
			}
			return result, nil
		}, func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Len(t, values, 3)
		require.Len(t, hits, 3)

		for _, key := range keys {
			require.Equal(t, cache.Hit, hits[key])
			require.Equal(t, "value_"+key, values[key])
		}
	})

	t.Run("all keys hit - no origin call", func(t *testing.T) {
		// Populate cache
		c.Set(ctx, "a", "va")
		c.Set(ctx, "b", "vb")

		keys := []string{"a", "b"}
		values, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			t.Fatal("should not call refresh function")
			return nil, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Len(t, values, 2)
		require.Len(t, hits, 2)

		require.Equal(t, cache.Hit, hits["a"])
		require.Equal(t, "va", values["a"])

		require.Equal(t, cache.Hit, hits["b"])
		require.Equal(t, "vb", values["b"])
	})

	t.Run("mixed hits and misses", func(t *testing.T) {
		// Populate some keys
		c.Set(ctx, "cached1", "value1")
		c.Set(ctx, "cached2", "value2")

		keys := []string{"cached1", "cached2", "new1", "new2"}
		fetchedKeys := []string{}

		values, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			fetchedKeys = keysToFetch
			result := make(map[string]string)
			for _, k := range keysToFetch {
				result[k] = "fetched_" + k
			}
			return result, nil
		}, func(err error) cache.Op {
			if err != nil {
				return cache.Noop
			}
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Len(t, values, 4)
		require.Len(t, hits, 4)

		// Cached keys should return cached values
		require.Equal(t, cache.Hit, hits["cached1"])
		require.Equal(t, "value1", values["cached1"])

		require.Equal(t, cache.Hit, hits["cached2"])
		require.Equal(t, "value2", values["cached2"])

		// New keys should be fetched
		require.Equal(t, cache.Hit, hits["new1"])
		require.Equal(t, "fetched_new1", values["new1"])

		require.Equal(t, cache.Hit, hits["new2"])
		require.Equal(t, "fetched_new2", values["new2"])

		// Verify only missing keys were fetched
		require.Len(t, fetchedKeys, 2)
		require.Contains(t, fetchedKeys, "new1")
		require.Contains(t, fetchedKeys, "new2")
	})

	t.Run("stale keys return cached value and refresh in background", func(t *testing.T) {
		// Populate cache
		c.Set(ctx, "stale1", "old_value1")
		c.Set(ctx, "stale2", "old_value2")

		// Move past fresh but within stale
		mockClock.Tick(2 * time.Minute)

		keys := []string{"stale1", "stale2"}
		values, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			// This will be called in background
			result := make(map[string]string)
			for _, k := range keysToFetch {
				result[k] = "new_" + k
			}
			return result, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Len(t, values, 2)

		// Should return old cached values
		require.Equal(t, "old_value1", values["stale1"])
		require.Equal(t, "old_value2", values["stale2"])

		// All should be hits
		require.Equal(t, cache.Hit, hits["stale1"])
		require.Equal(t, cache.Hit, hits["stale2"])
	})

	t.Run("null values", func(t *testing.T) {
		keys := []string{"notfound1", "notfound2"}
		_, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			return nil, sql.ErrNoRows
		}, func(err error) cache.Op {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}
			return cache.Noop
		})

		require.Error(t, err)
		require.True(t, db.IsNotFound(err))

		// Keys should be marked as null
		for _, key := range keys {
			require.Equal(t, cache.Null, hits[key])
		}

		// Second call should return null hits without calling origin
		values2, hits2, err2 := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			t.Fatal("should not call refresh function for null values")
			return nil, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err2)
		for _, key := range keys {
			require.Equal(t, cache.Null, hits2[key])
			require.Equal(t, "", values2[key])
		}
	})

	t.Run("partial null values", func(t *testing.T) {
		keys := []string{"found", "notfound"}
		_, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			// Return value for "found" but indicate not found overall
			return map[string]string{"found": "value"}, sql.ErrNoRows
		}, func(err error) cache.Op {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}
			return cache.WriteValue
		})

		require.Error(t, err)
		require.True(t, db.IsNotFound(err))

		// All keys should be marked as null when WriteNull is used
		for _, key := range keys {
			require.Equal(t, cache.Null, hits[key])
		}
	})

	t.Run("error handling", func(t *testing.T) {
		keys := []string{"err1", "err2"}
		expectedErr := fmt.Errorf("fetch error")

		_, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			return nil, expectedErr
		}, func(err error) cache.Op {
			return cache.Noop
		})

		require.Error(t, err)
		require.Equal(t, expectedErr, err)

		// Keys should be miss on error
		for _, key := range keys {
			require.Equal(t, cache.Miss, hits[key])
		}
	})

	t.Run("empty keys slice", func(t *testing.T) {
		values, hits, err := c.SWRMany(ctx, []string{}, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			t.Fatal("should not call refresh for empty keys")
			return nil, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)
		require.Len(t, values, 0)
		require.Len(t, hits, 0)
	})

	t.Run("deduplication - only fetch unique keys", func(t *testing.T) {
		// Pre-populate one key
		c.Set(ctx, "cached", "cached_value")

		keys := []string{"cached", "new", "new"} // duplicate "new"
		var fetchedKeys []string

		values, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			fetchedKeys = keysToFetch
			return map[string]string{"new": "new_value"}, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)

		// Should only fetch "new" once
		require.Len(t, fetchedKeys, 1)
		require.Equal(t, "new", fetchedKeys[0])

		// Should have results for unique keys only (maps deduplicate)
		require.Len(t, hits, 2)
		require.Len(t, values, 2)
		require.Equal(t, "cached_value", values["cached"])
		require.Equal(t, "new_value", values["new"])
	})

	t.Run("partial results - cache NULL for missing keys", func(t *testing.T) {
		keys := []string{"exists1", "exists2", "missing1", "missing2"}

		// First call - DB only returns 2 of 4 keys
		values, hits, err := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			// Simulate DB returning partial results
			return map[string]string{
				"exists1": "value1",
				"exists2": "value2",
				// "missing1" and "missing2" not returned
			}, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err)

		// The found keys should be Hit with values
		require.Equal(t, cache.Hit, hits["exists1"])
		require.Equal(t, "value1", values["exists1"])
		require.Equal(t, cache.Hit, hits["exists2"])
		require.Equal(t, "value2", values["exists2"])

		// The missing keys should be cached as Null (values map will have zero values)
		require.Equal(t, cache.Null, hits["missing1"])
		require.Equal(t, "", values["missing1"]) // zero value for string
		require.Equal(t, cache.Null, hits["missing2"])
		require.Equal(t, "", values["missing2"]) // zero value for string

		// Second call - should return cached values without calling origin
		values2, hits2, err2 := c.SWRMany(ctx, keys, func(ctx context.Context, keysToFetch []string) (map[string]string, error) {
			t.Fatal("should not call refresh - all keys should be cached")
			return nil, nil
		}, func(err error) cache.Op {
			return cache.WriteValue
		})

		require.NoError(t, err2)

		// All keys should be in cache now
		require.Equal(t, cache.Hit, hits2["exists1"])
		require.Equal(t, "value1", values2["exists1"])
		require.Equal(t, cache.Hit, hits2["exists2"])
		require.Equal(t, "value2", values2["exists2"])
		require.Equal(t, cache.Null, hits2["missing1"]) // Cached as null
		require.Equal(t, "", values2["missing1"])
		require.Equal(t, cache.Null, hits2["missing2"]) // Cached as null
		require.Equal(t, "", values2["missing2"])
	})
}
