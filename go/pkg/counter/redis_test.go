package counter

import (
	"context"
	"fmt"
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
		for i := range numWorkers {
			go func(id int) {
				defer wg.Done()

				for j := range operationsPerWorker {
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
		val, existed, success, err := ctr.DecrementIfExists(ctx, key, 5)
		require.NoError(t, err)
		require.False(t, existed, "Key should not exist")
		require.False(t, success, "Success should be false when key doesn't exist")
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
		val, existed, success, err := ctr.DecrementIfExists(ctx, key, 3)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist")
		require.True(t, success, "Decrement should succeed")
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

		// Attempt to decrement by more than current value - should be refused
		val, existed, success, err := ctr.DecrementIfExists(ctx, key, 8)
		require.NoError(t, err)
		require.True(t, existed, "Key should exist")
		require.False(t, success, "Decrement should fail due to insufficient credits")
		require.Equal(t, int64(3), val, "Should return actual current value when insufficient credits")

		// Verify the original value is unchanged
		currentVal, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(3), currentVal, "Original value should be unchanged after refused decrement")
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
				_, existed, success, err := ctr.DecrementIfExists(ctx, key, 1)
				if err != nil {
					t.Errorf("DecrementIfExists error: %v", err)
					return
				}
				if existed && success {
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

// TestRedisCounterDelete tests the Delete operation on the Redis counter.
func TestRedisCounterDelete(t *testing.T) {
	ctx := context.Background()
	logger := logging.New()
	redisURL := containers.Redis(t)

	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logger,
	})
	require.NoError(t, err)
	defer ctr.Close()

	t.Run("DeleteExistingKey", func(t *testing.T) {
		key := fmt.Sprintf("test-delete-%d", time.Now().UnixNano())

		// Set initial value
		val, err := ctr.Increment(ctx, key, 100)
		require.NoError(t, err)
		require.Equal(t, int64(100), val)

		// Verify it exists
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(100), val)

		// Delete the key
		err = ctr.Delete(ctx, key)
		require.NoError(t, err)

		// Verify it's gone (should return 0)
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), val)

		// DecrementIfExists should fail now
		val, existed, success, err := ctr.DecrementIfExists(ctx, key, 10)
		require.NoError(t, err)
		require.False(t, existed)
		require.False(t, success)
		require.Equal(t, int64(0), val)
	})

	t.Run("DeleteNonExistentKey", func(t *testing.T) {
		key := fmt.Sprintf("test-delete-nonexistent-%d", time.Now().UnixNano())

		// Delete a key that doesn't exist (should not error)
		err := ctr.Delete(ctx, key)
		require.NoError(t, err)

		// Verify it still doesn't exist
		val, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), val)
	})

	t.Run("DeleteAndReinitialize", func(t *testing.T) {
		key := fmt.Sprintf("test-delete-reinit-%d", time.Now().UnixNano())

		// Set initial value
		val, err := ctr.Increment(ctx, key, 50)
		require.NoError(t, err)
		require.Equal(t, int64(50), val)

		// Delete the key
		err = ctr.Delete(ctx, key)
		require.NoError(t, err)

		// Reinitialize with a new value
		val, err = ctr.Increment(ctx, key, 25)
		require.NoError(t, err)
		require.Equal(t, int64(25), val)

		// Verify the new value
		val, err = ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(25), val)
	})
}

// TestRedisCounterDecrementLogic tests the decrement logic that avoids negative values
func TestRedisCounterDecrementLogic(t *testing.T) {
	ctx := context.Background()
	logger := logging.New()
	redisURL := containers.Redis(t)

	ctr, err := NewRedis(RedisConfig{
		RedisURL: redisURL,
		Logger:   logger,
	})
	require.NoError(t, err)
	defer ctr.Close()

	t.Run("BasicDecrementLogic", func(t *testing.T) {
		key := fmt.Sprintf("test-decrement-%d", time.Now().UnixNano())

		// Initialize with 10 credits
		val, err := ctr.Increment(ctx, key, 10)
		require.NoError(t, err)
		require.Equal(t, int64(10), val)

		// Test cases: [decrementAmount, expectedSuccess, expectedFinalValue]
		testCases := []struct {
			decrementAmount    int64
			expectedSuccess    bool
			expectedFinalValue int64
			description        string
		}{
			{3, true, 7, "decrement 3 from 10 → success, 7 remaining"},
			{5, true, 2, "decrement 5 from 7 → success, 2 remaining"},
			{2, true, 0, "decrement 2 from 2 → success, 0 remaining"},
			{1, false, 0, "decrement 1 from 0 → failure, 0 remaining (unchanged)"},
			{5, false, 0, "decrement 5 from 0 → failure, 0 remaining (unchanged)"},
		}

		for _, tc := range testCases {
			t.Logf("Testing: %s", tc.description)

			remaining, existed, success, err := ctr.DecrementIfExists(ctx, key, tc.decrementAmount)
			require.NoError(t, err)
			require.True(t, existed, "key should exist")

			if tc.expectedSuccess {
				require.True(t, success, "decrement should succeed when sufficient credits")
				require.Equal(t, tc.expectedFinalValue, remaining, "successful decrement should return correct remaining value")
				require.GreaterOrEqual(t, remaining, int64(0), "successful decrement should never go negative")
			} else {
				require.False(t, success, "decrement should fail when insufficient credits")
				require.GreaterOrEqual(t, remaining, int64(0), "failed decrement should return actual current count")
			}

			// Verify actual counter value
			actualVal, err := ctr.Get(ctx, key)
			require.NoError(t, err)
			require.Equal(t, tc.expectedFinalValue, actualVal, "counter value should match expected")
		}
	})

	t.Run("ConcurrentDecrement", func(t *testing.T) {
		key := fmt.Sprintf("test-concurrent-decrement-%d", time.Now().UnixNano())

		// Initialize with 100 credits
		initialCredits := int64(100)
		_, err := ctr.Increment(ctx, key, initialCredits)
		require.NoError(t, err)

		// Run 50 goroutines, each trying to decrement 3 credits
		// Only first 33 should succeed (33 * 3 = 99), with 1 credit remaining

		const numGoroutines = 50
		const decrementAmount = 3
		const expectedSuccessful = 33 // floor(100/3) = 33

		// Start all goroutines simultaneously
		var wg sync.WaitGroup
		startBarrier := make(chan struct{})

		for range numGoroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Wait for all goroutines to be ready
				<-startBarrier

				remaining, existed, success, err := ctr.DecrementIfExists(ctx, key, decrementAmount)
				require.NoError(t, err)
				require.True(t, existed)
				require.GreaterOrEqual(t, remaining, int64(0), "decrement should always return non-negative actual count")

				// We don't need to check success here since this is a concurrent test
				// The final counter value verification is what matters
				_ = success
			}()
		}

		// Release all goroutines at once
		close(startBarrier)
		wg.Wait()

		// Verify final counter value - this is the real test of decrement accuracy
		finalValue, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		expectedFinalValue := initialCredits - int64(expectedSuccessful*decrementAmount)
		require.Equal(t, expectedFinalValue, finalValue, "decrement should result in exactly the correct final value")

		t.Logf(" decrement concurrency test: final value %d (expected %d)", finalValue, expectedFinalValue)
	})

	t.Run("ExactCreditUsage", func(t *testing.T) {
		key := fmt.Sprintf("test-exact-usage-%d", time.Now().UnixNano())

		// Initialize with exactly 15 credits
		_, err := ctr.Increment(ctx, key, 15)
		require.NoError(t, err)

		// Decrement exactly 15 credits - should succeed and leave 0
		remaining, existed, success, err := ctr.DecrementIfExists(ctx, key, 15)
		require.NoError(t, err)
		require.True(t, existed)
		require.True(t, success, "should succeed when using exact amount of credits")
		require.Equal(t, int64(0), remaining, "should be able to use exact amount of credits")

		// Next decrement should fail
		remaining, existed, success, err = ctr.DecrementIfExists(ctx, key, 1)
		require.NoError(t, err)
		require.True(t, existed)
		require.False(t, success, "should fail when no credits left")
		require.Equal(t, int64(0), remaining, "should return actual current count (0) when no credits left")

		// Verify counter is still 0
		finalValue, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), finalValue, "counter should remain at 0")
	})

	t.Run("HighConcurrencyEdgeCases", func(t *testing.T) {
		key := fmt.Sprintf("test-edge-cases-%d", time.Now().UnixNano())

		// Test with a small number of credits and high concurrency
		initialCredits := int64(5)
		_, err := ctr.Increment(ctx, key, initialCredits)
		require.NoError(t, err)

		const numGoroutines = 20
		const decrementAmount = 1

		results := make(chan bool, numGoroutines)
		var wg sync.WaitGroup
		startBarrier := make(chan struct{})

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-startBarrier

				remaining, existed, success, err := ctr.DecrementIfExists(ctx, key, decrementAmount)
				require.NoError(t, err)
				require.True(t, existed)
				require.GreaterOrEqual(t, remaining, int64(0), "should always return non-negative actual count")

				results <- success
			}()
		}

		close(startBarrier)
		wg.Wait()
		close(results)

		// Count successes
		var successCount int
		for success := range results {
			if success {
				successCount++
			}
		}

		t.Logf("High concurrency test: %d successes out of %d attempts", successCount, numGoroutines)

		// Should have exactly initialCredits successes
		require.Equal(t, int(initialCredits), successCount, "should have exactly %d successes", initialCredits)

		// Final value should be 0
		finalValue, err := ctr.Get(ctx, key)
		require.NoError(t, err)
		require.Equal(t, int64(0), finalValue, "final value should be 0")
	})
}
