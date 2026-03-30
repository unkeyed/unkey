package counter

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value  int64
	expiry time.Time // zero means no expiry
}

func (e memoryEntry) expired(now time.Time) bool {
	return !e.expiry.IsZero() && now.After(e.expiry)
}

type memoryCounter struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
}

// NewMemory creates a new in-memory counter.
// Entries with a TTL are lazily expired on access.
func NewMemory() Counter {
	//nolint:exhaustruct
	return &memoryCounter{
		entries: make(map[string]memoryEntry),
	}
}

func (m *memoryCounter) Increment(_ context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	e, ok := m.entries[key]
	if !ok || e.expired(now) {
		e = memoryEntry{value: 0, expiry: time.Time{}}
		if len(ttl) > 0 && ttl[0] > 0 {
			e.expiry = now.Add(ttl[0])
		}
	}

	e.value += value
	m.entries[key] = e
	return e.value, nil
}

func (m *memoryCounter) Get(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.entries[key]
	if !ok || e.expired(time.Now()) {
		delete(m.entries, key)
		return 0, nil
	}
	return e.value, nil
}

func (m *memoryCounter) MultiGet(_ context.Context, keys []string) (map[string]int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	result := make(map[string]int64, len(keys))
	for _, key := range keys {
		e, ok := m.entries[key]
		if ok && !e.expired(now) {
			result[key] = e.value
		} else {
			delete(m.entries, key)
			result[key] = 0
		}
	}
	return result, nil
}

func (m *memoryCounter) Decrement(_ context.Context, key string, value int64, ttl ...time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	e, ok := m.entries[key]
	if !ok || e.expired(now) {
		e = memoryEntry{value: 0, expiry: time.Time{}}
		if len(ttl) > 0 && ttl[0] > 0 {
			e.expiry = now.Add(ttl[0])
		}
	}

	e.value -= value
	m.entries[key] = e
	return e.value, nil
}

func (m *memoryCounter) DecrementIfExists(_ context.Context, key string, value int64) (int64, bool, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	e, ok := m.entries[key]
	if !ok || e.expired(time.Now()) {
		delete(m.entries, key)
		return 0, false, false, nil
	}

	if e.value < value {
		return e.value, true, false, nil
	}

	e.value -= value
	m.entries[key] = e
	return e.value, true, true, nil
}

func (m *memoryCounter) SetIfNotExists(_ context.Context, key string, value int64, ttl ...time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	e, ok := m.entries[key]
	if ok && !e.expired(now) {
		return false, nil
	}

	e = memoryEntry{value: value, expiry: time.Time{}}
	if len(ttl) > 0 && ttl[0] > 0 {
		e.expiry = now.Add(ttl[0])
	}
	m.entries[key] = e
	return true, nil
}

func (m *memoryCounter) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.entries, key)
	return nil
}

func (m *memoryCounter) Close() error {
	return nil
}
