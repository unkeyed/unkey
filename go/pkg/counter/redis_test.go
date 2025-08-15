package counter

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func TestRedisCounter(t *testing.T) {
	ctx := context.Background()
	redisURL := containers.Redis(t)

	// Create a Redis counter
	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logging.New(),
	})
	require.NoError(t, err)
	defer ctr.Close()

	// Test basic increment
	t.Run("BasicIncrement", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// First increment should return 1
		val, err := ctr.Increment(ctx, key, 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), val)

		// Second increment should return 2
		val, err = ctr.Increment(ctx, key, 1)
		require.NoError(t, err)
		require.Equal(t, int64(2), val)

		// Increment by 5 should return 7
		val, err = ctr.Increment(ctx, key, 5)
		require.NoError(t, err)
		require.Equal(t, int64(7), val)
	})

	t.Run("IncrementWithTTL", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		ttl := 1 * time.Second

		// First increment with TTL
		val, err := ctr.Increment(ctx, key, 1, ttl)
		require.NoError(t, err)
		require.Equal(t, int64(1), val)

		// Get the value immediately
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(1), val)

		// Wait for the key to expire
		time.Sleep(2 * time.Second)

		// Key should be gone or zero
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), val)
	})

	t.Run("Get", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Get non-existent key
		val, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), val)

		// Set a value and get it
		_, err = ctr.Increment(ctx, key, 42)
		require.NoError(t, err)

		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(42), val)
	})

	// Table-driven tests for multiple increments
	t.Run("TableDrivenIncrements", func(t *testing.T) {
		tests := []struct {
			name       string
			key        string
			increments []int64
			expected   int64
		}{
			{
				name:       "Single increment",
				increments: []int64{5},
				expected:   5,
			},
			{
				name:       "Multiple increments",
				increments: []int64{1, 2, 3, 4, 5},
				expected:   15,
			},
			{
				name:       "Mixed positive and negative",
				increments: []int64{10, -3, 5, -2},
				expected:   10,
			},
			{
				name:       "Zero sum",
				increments: []int64{5, -5, 10, -10},
				expected:   0,
			},
			{
				name:       "Large increments",
				increments: []int64{1000, 2000, 3000},
				expected:   6000,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				key := uid.New(uid.TestPrefix)
				var finalValue int64
				var err error

				for _, inc := range tc.increments {
					finalValue, err = ctr.Increment(ctx, key, inc)
					require.NoError(t, err)
				}

				require.Equal(t, tc.expected, finalValue)

				// Verify with Get also
				value, err := ctr.Get(ctx, key)
				require.NoError(t, err)
				require.Equal(t, tc.expected, value)
			})
		}
	})

	// Test concurrent increments
	t.Run("ConcurrentIncrements", func(t *testing.T) {
		tests := []struct {
			name           string
			goroutines     int
			incrementsEach int
			value          int64
			expected       int64
		}{
			{
				name:           "Few goroutines, many increments",
				goroutines:     5,
				incrementsEach: 100,
				value:          1,
				expected:       500, // 5 * 100 * 1
			},
			{
				name:           "Many goroutines, few increments",
				goroutines:     50,
				incrementsEach: 10,
				value:          1,
				expected:       500, // 50 * 10 * 1
			},
			{
				name:           "Medium scale mixed values",
				goroutines:     20,
				incrementsEach: 20,
				value:          5,
				expected:       2000, // 20 * 20 * 5
			},
			{
				name:           "High contention with negative values",
				goroutines:     30,
				incrementsEach: 10,
				value:          -2,
				expected:       -600, // 30 * 10 * -2
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				var wg sync.WaitGroup
				wg.Add(tc.goroutines)

				key := uid.New(uid.TestPrefix)

				for i := 0; i < tc.goroutines; i++ {
					go func() {
						defer wg.Done()
						for j := 0; j < tc.incrementsEach; j++ {
							_, err := ctr.Increment(ctx, key, tc.value)
							if err != nil {
								t.Errorf("increment error: %v", err)
								return
							}
						}
					}()
				}

				wg.Wait()

				// Verify final value
				value, err := ctr.Get(ctx, key)
				require.NoError(t, err)
				require.Equal(t, tc.expected, value, "Final counter value doesn't match expected")
			})
		}
	})

	// Test interleaved operations (increment and get mixed together)
	t.Run("InterleavedOperations", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		numWorkers := 10
		operationsPerWorker := 50

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		// Launch goroutines that both increment and get values
		for i := 0; i < numWorkers; i++ {
			go func(id int) {
				defer wg.Done()

				for j := 0; j < operationsPerWorker; j++ {
					// Alternate between increment and get
					if j%2 == 0 {
						_, err := ctr.Increment(ctx, key, 1)
						if err != nil {
							t.Errorf("worker %d: increment error: %v", id, err)
							return
						}
					} else {
						_, err := ctr.Get(ctx, key)
						if err != nil {
							t.Errorf("worker %d: get error: %v", id, err)
							return
						}
					}
				}
			}(i)
		}

		wg.Wait()

		// Calculate expected value: each worker does operationsPerWorker/2 increments
		expectedValue := int64(numWorkers * (operationsPerWorker / 2))

		// Verify final value
		value, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, expectedValue, value, "Final value after interleaved operations doesn't match expected")
	})

	// Test increments with TTL in parallel
	t.Run("ConcurrentTTLIncrements", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		numWorkers := 10
		ttl := 3 * time.Second

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func() {
				defer wg.Done()
				_, err := ctr.Increment(ctx, key, 1, ttl)
				if err != nil {
					t.Errorf("increment with TTL error: %v", err)
				}
			}()
		}

		wg.Wait()

		// Verify value right after increments
		value, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(numWorkers), value)

		// Wait for TTL to expire
		time.Sleep(4 * time.Second)

		// Key should be gone or zero
		value, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), value, "Counter should be zero after TTL expiry")
	})
}

func TestRedisCounterConnection(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		// Test with invalid URL
		_, err := NewRedis(RedisConfig{
			RedisURL: "invalid://url",
			Logger:   logging.New(),
		})
		require.Error(t, err)
	})

	t.Run("ConnectionRefused", func(t *testing.T) {
		// Test with non-existent Redis server
		_, err := NewRedis(RedisConfig{
			RedisURL: "redis://localhost:12345",
			Logger:   logging.New(),
		})
		require.Error(t, err)
	})

	t.Run("EmptyURL", func(t *testing.T) {
		// Test with empty URL
		_, err := NewRedis(RedisConfig{
			RedisURL: "",
			Logger:   logging.New(),
		})
		require.Error(t, err)
	})
}

func TestRedisCounterMultiGet(t *testing.T) {
	ctx := context.Background()
	redisURL := containers.Redis(t)

	// Create a Redis counter
	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logging.New(),
	})
	require.NoError(t, err)
	defer ctr.Close()

	// Set up some test data
	testData := map[string]int64{
		uid.New(uid.TestPrefix): 10,
		uid.New(uid.TestPrefix): 20,
		uid.New(uid.TestPrefix): 30,
		uid.New(uid.TestPrefix): 40,
		uid.New(uid.TestPrefix): 50,
	}

	// Initialize counters
	for key, value := range testData {
		_, err := ctr.Increment(ctx, key, value)
		require.NoError(t, err)
	}

	t.Run("MultiGetAllExisting", func(t *testing.T) {
		keys := []string{}
		for key := range testData {
			keys = append(keys, key)
		}
		values, err := ctr.MultiGet(ctx, keys)
		require.NoError(t, err)

		// Verify all values match expected
		for key, expectedValue := range testData {
			value, exists := values[key]
			require.True(t, exists, "Key %s should exist in results", key)
			require.Equal(t, expectedValue, value, "Value for key %s should match", key)
		}
	})

	t.Run("MultiGetEmpty", func(t *testing.T) {
		values, err := ctr.MultiGet(ctx, []string{})
		require.NoError(t, err)
		require.Empty(t, values, "Result should be empty for empty keys list")
	})

	t.Run("MultiGetNonExisting", func(t *testing.T) {
		keys := []string{
			uid.New(uid.TestPrefix),
			uid.New(uid.TestPrefix),
			uid.New(uid.TestPrefix),
		}
		values, err := ctr.MultiGet(ctx, keys)
		require.NoError(t, err)

		// All values should be 0
		for _, key := range keys {
			require.Equal(t, int64(0), values[key])
		}
	})

	t.Run("MultiGetLarge", func(t *testing.T) {
		// Set up 100 counters
		largeTestData := make(map[string]int64)
		var largeKeys []string

		for i := 0; i < 100; i++ {
			key := uid.New(uid.TestPrefix)
			largeTestData[key] = int64(i)
			largeKeys = append(largeKeys, key)
			_, err := ctr.Increment(ctx, key, int64(i))
			require.NoError(t, err)
		}

		// Get all values
		values, err := ctr.MultiGet(ctx, largeKeys)
		require.NoError(t, err)

		// Verify all values match expected
		require.Equal(t, len(largeTestData), len(values))
		for key, expectedValue := range largeTestData {
			require.Equal(t, expectedValue, values[key])
		}
	})

	t.Run("ConcurrentMultiGet", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		keys := []string{
			uid.New(uid.TestPrefix),
			uid.New(uid.TestPrefix),
			uid.New(uid.TestPrefix),
			uid.New(uid.TestPrefix),
			uid.New(uid.TestPrefix),
		}

		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					values, err := ctr.MultiGet(ctx, keys)
					if err != nil {
						t.Errorf("MultiGet error: %v", err)
						return
					}

					// Verify key counts
					if len(values) != len(keys) {
						t.Errorf("Expected %d values, got %d", len(keys), len(values))
					}
				}
			}()
		}

		wg.Wait()
	})
}

func TestRedisCounterDecrement(t *testing.T) {
	ctx := context.Background()
	redisURL := containers.Redis(t)

	// Create a Redis counter
	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logging.New(),
	})
	require.NoError(t, err)
	defer ctr.Close()

	t.Run("BasicDecrement", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Start with initial value
		_, err := ctr.Increment(ctx, key, 10)
		require.NoError(t, err)

		// Decrement by 3 should return 7
		val, err := ctr.Decrement(ctx, key, 3)
		require.NoError(t, err)
		require.Equal(t, int64(7), val)

		// Decrement by 5 should return 2
		val, err = ctr.Decrement(ctx, key, 5)
		require.NoError(t, err)
		require.Equal(t, int64(2), val)

		// Decrement by 10 should return -8 (negative values allowed)
		val, err = ctr.Decrement(ctx, key, 10)
		require.NoError(t, err)
		require.Equal(t, int64(-8), val)
	})

	t.Run("DecrementWithTTL", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		ttl := 1 * time.Second

		// First decrement creates the key with TTL
		val, err := ctr.Decrement(ctx, key, 5, ttl)
		require.NoError(t, err)
		require.Equal(t, int64(-5), val) // Starting from 0, decrement by 5 = -5

		// Get the value immediately
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(-5), val)

		// Wait for the key to expire
		time.Sleep(2 * time.Second)

		// Key should be gone or zero
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), val)
	})

	t.Run("ConcurrentDecrements", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)
		numWorkers := 10
		decrementsPerWorker := 5

		// Start with a high value
		_, err := ctr.Increment(ctx, key, 1000)
		require.NoError(t, err)

		var wg sync.WaitGroup
		wg.Add(numWorkers)

		for i := 0; i < numWorkers; i++ {
			go func() {
				defer wg.Done()
				for j := 0; j < decrementsPerWorker; j++ {
					_, err := ctr.Decrement(ctx, key, 2)
					if err != nil {
						t.Errorf("decrement error: %v", err)
						return
					}
				}
			}()
		}

		wg.Wait()

		// Expected: 1000 - (10 workers * 5 decrements * 2) = 1000 - 100 = 900
		value, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(900), value)
	})
}

func TestRedisCounterDecrementIfExists(t *testing.T) {
	ctx := context.Background()
	redisURL := containers.Redis(t)

	// Create a Redis counter
	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logging.New(),
	})
	require.NoError(t, err)
	defer ctr.Close()

	t.Run("DecrementNonExistentKey", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Try to decrement a non-existent key
		val, existed, err := ctr.DecrementIfExists(ctx, key, 5)
		require.NoError(t, err)
		require.False(t, existed, "Key should not exist")
		require.Equal(t, int64(0), val, "Value should be 0 when key doesn't exist")

		// Verify key still doesn't exist
		getVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), getVal)
	})

	t.Run("DecrementExistingKey", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Create the key first
		_, err := ctr.Increment(ctx, key, 10)
		require.NoError(t, err)

		// Now decrement if exists - should work
		val, existed, err := ctr.DecrementIfExists(ctx, key, 3)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist")
		require.Equal(t, int64(7), val, "Value should be 7 after decrement")

		// Verify with Get
		getVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(7), getVal)
	})

	t.Run("DecrementToNegative", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Create key with small positive value
		_, err := ctr.Increment(ctx, key, 3)
		require.NoError(t, err)

		// Decrement by more than current value
		val, existed, err := ctr.DecrementIfExists(ctx, key, 8)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist")
		require.Equal(t, int64(-5), val, "Value should be -5 after decrement")
	})

	t.Run("ConcurrentDecrementIfExists", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Start with a value
		_, err := ctr.Increment(ctx, key, 100)
		require.NoError(t, err)

		numWorkers := 20
		var wg sync.WaitGroup
		var successCount int64
		var mu sync.Mutex

		wg.Add(numWorkers)
		for i := 0; i < numWorkers; i++ {
			go func() {
				defer wg.Done()
				_, existed, err := ctr.DecrementIfExists(ctx, key, 1)
				if err != nil {
					t.Errorf("DecrementIfExists error: %v", err)
					return
				}
				if existed {
					mu.Lock()
					successCount++
					mu.Unlock()
				}
			}()
		}

		wg.Wait()

		// All should have succeeded since key existed
		require.Equal(t, int64(numWorkers), successCount)

		// Final value should be 100 - 20 = 80
		finalVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(80), finalVal)
	})
}

func TestRedisCounterIncrementIfExists(t *testing.T) {
	ctx := context.Background()
	redisURL := containers.Redis(t)

	// Create a Redis counter
	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logging.New(),
	})
	require.NoError(t, err)
	defer ctr.Close()

	t.Run("IncrementNonExistentKey", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Try to increment a non-existent key
		val, existed, err := ctr.IncrementIfExists(ctx, key, 5)
		require.NoError(t, err)
		require.False(t, existed, "Key should not exist")
		require.Equal(t, int64(0), val, "Value should be 0 when key doesn't exist")

		// Verify key still doesn't exist
		getVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), getVal)
	})

	t.Run("IncrementExistingKey", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Create the key first
		_, err := ctr.Increment(ctx, key, 5)
		require.NoError(t, err)

		// Now increment if exists - should work
		val, existed, err := ctr.IncrementIfExists(ctx, key, 3)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist")
		require.Equal(t, int64(8), val, "Value should be 8 after increment")

		// Verify with Get
		getVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(8), getVal)
	})

	t.Run("IncrementWithNegativeValue", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Create key with positive value
		_, err := ctr.Increment(ctx, key, 10)
		require.NoError(t, err)

		// Increment with negative value (effectively decrement)
		val, existed, err := ctr.IncrementIfExists(ctx, key, -3)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist")
		require.Equal(t, int64(7), val, "Value should be 7 after negative increment")
	})

	t.Run("UsageLimiterScenario", func(t *testing.T) {
		// This tests the specific scenario used in the usage limiter
		key := uid.New(uid.TestPrefix)

		// Simulate a key that doesn't exist (new user/key)
		remaining, existed, err := ctr.DecrementIfExists(ctx, key, 1)
		require.NoError(t, err)
		require.False(t, existed, "Key should not exist initially")
		require.Equal(t, int64(0), remaining)

		// Now create the key (simulating DB load) and set credits
		_, err = ctr.Increment(ctx, key, 5) // User has 5 credits
		require.NoError(t, err)

		// Try decrementing again - should work now
		remaining, existed, err = ctr.DecrementIfExists(ctx, key, 1)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist now")
		require.Equal(t, int64(4), remaining, "Should have 4 credits left")

		// Multiple decrements to test insufficient credits
		for i := 0; i < 4; i++ {
			remaining, existed, err = ctr.DecrementIfExists(ctx, key, 1)
			require.NoError(t, err)
			require.True(t, existed)
		}
		require.Equal(t, int64(0), remaining, "Should have 0 credits left")

		// Try one more decrement - should result in negative
		remaining, existed, err = ctr.DecrementIfExists(ctx, key, 1)
		require.NoError(t, err)
		require.True(t, existed, "Key should still exist")
		require.Equal(t, int64(-1), remaining, "Should be -1 (insufficient)")
	})
}

func TestRedisCounterMixedOperations(t *testing.T) {
	ctx := context.Background()
	redisURL := containers.Redis(t)

	// Create a Redis counter
	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logging.New(),
	})
	require.NoError(t, err)
	defer ctr.Close()

	t.Run("IncrementAndDecrementMixed", func(t *testing.T) {
		key := uid.New(uid.TestPrefix)

		// Start with increment
		val, err := ctr.Increment(ctx, key, 10)
		require.NoError(t, err)
		require.Equal(t, int64(10), val)

		// Decrement
		val, err = ctr.Decrement(ctx, key, 3)
		require.NoError(t, err)
		require.Equal(t, int64(7), val)

		// IncrementIfExists
		val, existed, err := ctr.IncrementIfExists(ctx, key, 5)
		require.NoError(t, err)
		require.True(t, existed)
		require.Equal(t, int64(12), val)

		// DecrementIfExists
		val, existed, err = ctr.DecrementIfExists(ctx, key, 2)
		require.NoError(t, err)
		require.True(t, existed)
		require.Equal(t, int64(10), val)

		// Final verification
		finalVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(10), finalVal)
	})
}
