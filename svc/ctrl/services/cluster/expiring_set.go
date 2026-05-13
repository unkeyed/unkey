package cluster

import (
	"sync"
	"time"
)

// expiringSet is a thread-safe set that drops entries after a TTL when
// Sweep is called. AddIfAbsent is the primary operation: it inserts a key
// the first time it's seen and returns false on subsequent calls, dedup'ing
// fire-and-forget side effects (e.g. "notify Restate exactly once per
// deployment").
//
// Currently lives inside the cluster package; if a second use case shows up
// it can be lifted out to pkg/expiringset without API changes.
type expiringSet[K comparable] struct {
	mu    sync.Mutex
	items map[K]time.Time
	ttl   time.Duration
}

func newExpiringSet[K comparable](ttl time.Duration) *expiringSet[K] {
	return &expiringSet[K]{
		mu:    sync.Mutex{},
		items: make(map[K]time.Time),
		ttl:   ttl,
	}
}

// AddIfAbsent inserts the key if it isn't already present and returns true.
// Returns false if the key was already in the set (i.e. another caller has
// already added it within the TTL window).
func (s *expiringSet[K]) AddIfAbsent(key K) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.items[key]; exists {
		return false
	}
	s.items[key] = time.Now()
	return true
}

// Sweep drops entries older than the TTL. Returns the number dropped.
// Intended to be called periodically (e.g. via repeat.Every).
func (s *expiringSet[K]) Sweep() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-s.ttl)
	dropped := 0
	for k, t := range s.items {
		if t.Before(cutoff) {
			delete(s.items, k)
			dropped++
		}
	}
	return dropped
}
