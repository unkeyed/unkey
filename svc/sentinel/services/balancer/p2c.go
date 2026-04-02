package balancer

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"
)

var (
	_ Balancer        = (*P2CBalancer)(nil)
	_ InflightTracker = (*P2CBalancer)(nil)
)

// P2CBalancer implements Power of Two Choices load balancing with in-flight
// request tracking. It picks two random instances and routes to the one with
// fewer in-flight requests. This provides near-optimal load distribution with
// O(1) selection time.
//
// When there is only one instance, it is always returned. When there are two,
// both are compared directly. For three or more, two are sampled at random.
type P2CBalancer struct {
	mu      sync.RWMutex
	entries map[string]*atomic.Int64 // instanceID -> inflight count
}

func NewP2CBalancer() *P2CBalancer {
	return &P2CBalancer{
		mu:      sync.RWMutex{},
		entries: make(map[string]*atomic.Int64),
	}
}

func (b *P2CBalancer) Pick(instanceIDs []string) int {
	n := len(instanceIDs)
	if n == 1 {
		return 0
	}

	//nolint:gosec
	i := rand.IntN(n)
	//nolint:gosec
	j := rand.IntN(n - 1)
	if j >= i {
		j++
	}

	loadI := b.Inflight(instanceIDs[i])
	loadJ := b.Inflight(instanceIDs[j])

	if loadI <= loadJ {
		return i
	}
	return j
}

func (b *P2CBalancer) Acquire(instanceID string) {
	b.getOrCreate(instanceID).Add(1)
}

func (b *P2CBalancer) Release(instanceID string) {
	b.getOrCreate(instanceID).Add(-1)
}

// Inflight returns the current in-flight count for the given instance.
func (b *P2CBalancer) Inflight(instanceID string) int64 {
	b.mu.RLock()
	counter, ok := b.entries[instanceID]
	b.mu.RUnlock()
	if !ok {
		return 0
	}
	return counter.Load()
}

func (b *P2CBalancer) getOrCreate(instanceID string) *atomic.Int64 {
	b.mu.RLock()
	counter, ok := b.entries[instanceID]
	b.mu.RUnlock()
	if ok {
		return counter
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	// Double-check after acquiring write lock.
	if counter, ok = b.entries[instanceID]; ok {
		return counter
	}
	counter = &atomic.Int64{}
	b.entries[instanceID] = counter
	return counter
}
