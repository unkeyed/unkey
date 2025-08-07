package counter

import (
	"context"
	"sync"
	"time"
)

// memoryCounter implements the Counter interface using in-memory storage.
// It provides counter functionality backed by a simple map with mutex protection.
// This implementation is suitable for single-instance deployments or testing.
type memoryCounter struct {
	// mu protects the counters map from concurrent access
	mu sync.RWMutex
	// counters stores the counter values
	counters map[string]int64
	// expiry tracks when counters should expire (optional TTL support)
	expiry map[string]time.Time
}

var _ Counter = (*memoryCounter)(nil)

// NewMemory creates a new in-memory counter implementation.
//
// Returns:
//   - Counter: In-memory implementation of the Counter interface
func NewMemory() Counter {
	return &memoryCounter{
		counters: make(map[string]int64),
		expiry:   make(map[string]time.Time),
	}
}

// Increment increases the counter by the given value and returns the new count.
// If TTL is provided and the counter is newly created, it sets an expiration time.
//
// Parameters:
//   - ctx: Context for cancellation (unused in memory implementation)
//   - key: Unique identifier for the counter
//   - value: Amount to increment the counter by
//   - ttl: Optional time-to-live duration. If provided and key is new, sets TTL.
//
// Returns:
//   - int64: The new counter value after incrementing
//   - error: Always nil for memory implementation
func (m *memoryCounter) Increment(ctx context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clean up expired counters before processing
	m.cleanupExpired()

	// Get current value (0 if doesn't exist)
	currentValue := m.counters[key]
	isNewKey := currentValue == 0 && m.counters[key] == 0

	// Check if key exists in map to differentiate between new key and existing key with value 0
	_, exists := m.counters[key]
	if !exists {
		isNewKey = true
	}

	// Increment the counter
	newValue := currentValue + value
	m.counters[key] = newValue

	// Set TTL if provided and this is a new key
	if len(ttl) > 0 && isNewKey {
		m.expiry[key] = time.Now().Add(ttl[0])
	}

	return newValue, nil
}

// Get retrieves the current value of a counter.
//
// Parameters:
//   - ctx: Context for cancellation (unused in memory implementation)
//   - key: Unique identifier for the counter
//
// Returns:
//   - int64: The current counter value (0 if doesn't exist or expired)
//   - error: Always nil for memory implementation
func (m *memoryCounter) Get(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if counter has expired
	if expTime, hasExpiry := m.expiry[key]; hasExpiry {
		if time.Now().After(expTime) {
			// Counter has expired, return 0
			return 0, nil
		}
	}

	value, exists := m.counters[key]
	if !exists {
		return 0, nil
	}

	return value, nil
}

// MultiGet retrieves the values of multiple counters in a single operation.
//
// Parameters:
//   - ctx: Context for cancellation (unused in memory implementation)
//   - keys: Slice of unique identifiers for the counters
//
// Returns:
//   - map[string]int64: Map of counter keys to their current values
//   - error: Always nil for memory implementation
func (m *memoryCounter) MultiGet(ctx context.Context, keys []string) (map[string]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]int64, len(keys))
	now := time.Now()

	for _, key := range keys {
		// Check if counter has expired
		if expTime, hasExpiry := m.expiry[key]; hasExpiry {
			if now.After(expTime) {
				// Counter has expired, set to 0
				result[key] = 0
				continue
			}
		}

		value, exists := m.counters[key]
		if !exists {
			result[key] = 0
		} else {
			result[key] = value
		}
	}

	return result, nil
}

// Close releases any resources held by the counter implementation.
// For memory implementation, this just clears the internal maps.
//
// Returns:
//   - error: Always nil for memory implementation
func (m *memoryCounter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all data
	m.counters = make(map[string]int64)
	m.expiry = make(map[string]time.Time)

	return nil
}

// cleanupExpired removes expired counters from memory.
// This method should be called while holding a write lock.
func (m *memoryCounter) cleanupExpired() {
	now := time.Now()

	for key, expTime := range m.expiry {
		if now.After(expTime) {
			delete(m.counters, key)
			delete(m.expiry, key)
		}
	}
}
