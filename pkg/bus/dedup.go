package bus

import (
	"container/list"
	"sync"
)

// dedupKey is the composite (sender, event id) tuple the bus uses to detect
// duplicates. Keeping these as separate strings (rather than a concatenated
// key) avoids a tiny allocation per check on the hot dispatch path.
type dedupKey struct {
	sender string
	id     string
}

// dedupCache is a bounded set with FIFO eviction. It is an LRU in the sense
// that we re-touch on every duplicate seen, so a hot key stays in the cache
// for as long as duplicates keep arriving — the typical pattern under gossip
// retransmission.
type dedupCache struct {
	mu    sync.Mutex
	cap   int
	index map[dedupKey]*list.Element
	order *list.List
}

func newDedupCache(capacity int) *dedupCache {
	if capacity < 1 {
		capacity = 1
	}
	return &dedupCache{
		mu:    sync.Mutex{},
		cap:   capacity,
		index: make(map[dedupKey]*list.Element, capacity),
		order: list.New(),
	}
}

// SeenOrAdd returns true when the key has been seen before. New keys are
// inserted at the back; under capacity pressure the front (oldest) is
// evicted. On repeat hits we re-touch by moving the element to the back so a
// key that keeps re-appearing keeps deduping.
func (d *dedupCache) SeenOrAdd(k dedupKey) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if e, ok := d.index[k]; ok {
		d.order.MoveToBack(e)
		return true
	}

	e := d.order.PushBack(k)
	d.index[k] = e

	if d.order.Len() > d.cap {
		front := d.order.Front()
		if front != nil {
			d.order.Remove(front)
			delete(d.index, front.Value.(dedupKey))
		}
	}

	return false
}
