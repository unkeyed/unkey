// Package keymutex provides a keyed mutex used to serialize work per key.
//
// Krane's pod watch handles events concurrently (bounded by conc.Sem) so
// two events for the same k8s_name can race the test-and-set inside
// reportIfChanged. A keyed mutex serializes that window per k8s_name so
// each event observes a consistent fingerprint.
package keymutex

import "sync"

// KeyMutex issues per-key locks. Zero value is ready to use.
//
// Mutex entries are never cleaned up. That's intentional: the cardinality
// is bounded by the number of distinct keys the caller ever uses (e.g.
// k8s object names), the entries are a few dozen bytes each, and cleanup
// would require reference counting that adds more complexity than the
// memory saved.
type KeyMutex struct {
	mu sync.Map // string -> *sync.Mutex
}

// Lock acquires the per-key mutex and returns a function that releases it.
// Callers should `defer unlock()` immediately after the call.
func (k *KeyMutex) Lock(key string) (unlock func()) {
	v, _ := k.mu.LoadOrStore(key, &sync.Mutex{})
	m := v.(*sync.Mutex)
	m.Lock()
	return m.Unlock
}
