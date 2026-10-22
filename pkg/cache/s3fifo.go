package cache

// This file implements S3-FIFO, the eviction policy behind the cache. If you
// know a map-based cache with LRU eviction, here is what changes and why.
//
// The problem. A full cache must forget one key for every new key. LRU forgets
// the key that was used least recently. To know that, every read moves its
// entry to the front of a shared list, so every read writes shared state and
// needs a lock. LRU also has a weakness: most keys in real traffic are read
// once and never again ("one-hit wonders"). Each of them looks recently used,
// so a burst of them pushes out the keys that are actually hot.
//
// The idea. S3-FIFO never moves an entry on a read. A read only bumps a small
// counter on the entry, capped at frequencyMax. All decisions happen later,
// when a write needs room, by looking at those counters. It uses three FIFO
// queues, where new entries go in at the head and the oldest leave at the tail:
//
//	small  targets 10% of capacity. Every new key starts here, on probation.
//	main   the rest of capacity. Keys that were read again while in small.
//
// small and main together hold at most capacity entries. ghost is sized like
// main but stores keys only, so it costs little memory.
//	ghost  keys only, no values. Keys that small recently evicted.
//
// Eviction, step by step. When the cache is full, evict looks at the oldest
// entry of small (as long as small holds at least its 10% share):
//
//	read more than once  -> not a one-hit wonder; move it to main, reset its
//	                        counter, and keep looking.
//	read at most once    -> evict it and remember its key in ghost.
//
// When small is below its share, evict looks at the oldest entry of main:
//
//	counter above 0      -> it was read since its last turn; put it back at the
//	                        head of main with the counter decremented, and keep
//	                        looking. This is the "second chance" of CLOCK.
//	counter at 0         -> evict it.
//
// The ghost queue corrects mistakes. If a key comes back while ghost still
// remembers it, small evicted it too early, so set puts it straight into main.
//
// Example with capacity 10, so small's share is 1. Keys k0..k9 fill small, and
// k0 is read twice. Writing "new" then does this:
//
//	small (oldest first): k0 k1 ... k9    main: -
//	k0 was read twice  -> move k0 to main
//	k1 was never read  -> evict k1, remember k1 in ghost
//	small: k2 ... k9 new                  main: k0
//
// A plain FIFO would have evicted k0, the one key in use.
//
// Why this suits concurrency. Because reads never touch the queues, the only
// shared write on a hit is one atomic counter bump, and a key already at
// frequencyMax causes none. Entries live in a sync.Map, so lookups take no
// lock. The queues change only inside set, delete, and clear, under one mutex.
//
// Why this suits simulation. Eviction runs inside the set that needs room, on
// the caller's goroutine, and never uses randomness. The same sequence of
// calls therefore always evicts the same keys.

import (
	"iter"
	"sync"
	"sync/atomic"
)

const (
	// frequencyMax caps the read counter, so a key that was hot once loses its
	// protection after a few trips through the main queue without reads.
	frequencyMax = 3
	// reinsertionsMax bounds how many main-queue entries one eviction may
	// give a second chance before it evicts the next one.
	reinsertionsMax = 20
)

// node is one entry in a queue. Only frequency changes while other goroutines can
// read the node; key and value are fixed once the node is published.
type node[K comparable, V any] struct {
	key        K
	value      V
	frequency  atomic.Int32
	queue      *queue[K, V]
	prev, next *node[K, V]
}

// queue is a doubly linked FIFO. push adds at the head and pop removes from
// the tail, so the tail is always the oldest entry.
type queue[K comparable, V any] struct {
	head, tail *node[K, V]
	len        int
}

func (q *queue[K, V]) push(n *node[K, V]) {
	n.queue = q
	n.prev = nil //faultline:equivalent every pushed node is new or was unlinked, so prev is already nil
	n.next = q.head
	if q.head != nil {
		q.head.prev = n
	} else {
		q.tail = n
	}
	q.head = n
	q.len++
}

// remove unlinks n from anywhere in q.
func (q *queue[K, V]) remove(n *node[K, V]) {
	if n.prev != nil {
		n.prev.next = n.next
	} else {
		q.head = n.next
	}
	if n.next != nil {
		n.next.prev = n.prev
	} else {
		q.tail = n.prev
	}
	n.prev, n.next, n.queue = nil, nil, nil //faultline:equivalent unlinked nodes are relinked by push or replace before any read; clearing only helps the GC
	q.len--
}

// pop removes and returns the oldest entry. q must not be empty.
func (q *queue[K, V]) pop() *node[K, V] {
	n := q.tail
	q.remove(n)
	return n
}

// replace puts n at old's position in the queue.
func (q *queue[K, V]) replace(old, n *node[K, V]) {
	n.queue, n.prev, n.next = q, old.prev, old.next
	if n.prev != nil {
		n.prev.next = n
	} else {
		q.head = n
	}
	if n.next != nil {
		n.next.prev = n
	} else {
		q.tail = n
	}
	old.prev, old.next, old.queue = nil, nil, nil //faultline:equivalent unlinked nodes are relinked by push or replace before any read; clearing only helps the GC
}

// store is a fixed-capacity map that evicts with S3-FIFO
// (Yang et al., "FIFO queues are all you need for cache eviction", SOSP 2023).
// See the comment at the top of this file for how the policy works.
//
// entries holds every live node and is the only field reads touch. The other
// fields are the policy's bookkeeping and change only under mutex. ghosts
// indexes the ghost queue by key, so set can ask "was this key evicted
// recently?" in O(1).
type store[K comparable, V any] struct {
	entries sync.Map

	mutex  sync.Mutex
	ghosts map[K]*node[K, V]
	small  queue[K, V]
	main   queue[K, V]
	ghost  queue[K, V]
	// capacity bounds small.len + main.len, the number of live entries. The
	// ghost queue holds no values and does not count toward it.
	capacity int
	// capacitySmall is small's share of capacity. It is a target, not a hard
	// limit: new keys always enter small, so small can grow past it while main
	// is short of its share. evict brings it back by evicting from small first
	// whenever small holds capacitySmall or more. main has no field of its
	// own; it gets whatever capacity small does not use.
	capacitySmall int
}

// newStore returns an empty store that holds at most capacity entries.
func newStore[K comparable, V any](capacity int) *store[K, V] {
	s := new(store[K, V])
	s.ghosts = make(map[K]*node[K, V])
	s.capacity = capacity
	s.capacitySmall = max(1, capacity/10)
	return s
}

// get returns the value for key and counts the read toward its frequency.
func (s *store[K, V]) get(key K) (V, bool) {
	v, ok := s.entries.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	n := v.(*node[K, V])
	// Load before CompareAndSwap so a hot key at frequencyMax causes no write.
	// A lost race only drops one count, which the policy tolerates.
	if f := n.frequency.Load(); f < frequencyMax {
		n.frequency.CompareAndSwap(f, f+1)
	}
	return n.value, true
}

// set writes key and reports whether it replaced a value or evicted another
// entry to make room. Both answers fall out of work set does anyway.
func (s *store[K, V]) set(key K, value V) (replaced, evicted bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	n := new(node[K, V])
	n.key, n.value = key, value
	// An update builds a new node instead of writing old.value, because
	// readers may be reading old right now without a lock. The new node takes
	// old's place in its queue and keeps its counter, so an update neither
	// resets the key's eviction order nor counts as a read.
	if v, ok := s.entries.Load(key); ok {
		old := v.(*node[K, V])
		n.frequency.Store(old.frequency.Load())
		old.queue.replace(old, n)
		s.entries.Store(key, n)
		return true, false
	}
	// Live entries never exceed capacity and an insert adds one, so one
	// eviction always makes room.
	if s.small.len+s.main.len >= s.capacity {
		s.evict()
		evicted = true
	}
	// A key that small evicted recently has proven it comes back, so it skips
	// probation and goes straight to main.
	if g, ok := s.ghosts[key]; ok {
		s.ghost.remove(g)
		delete(s.ghosts, key)
		s.main.push(n)
	} else {
		s.small.push(n)
	}
	s.entries.Store(key, n)
	return false, evicted
}

// delete removes key and reports whether it was present.
func (s *store[K, V]) delete(key K) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v, ok := s.entries.LoadAndDelete(key)
	if !ok {
		return false
	}
	n := v.(*node[K, V])
	n.queue.remove(n)
	return true
}

// clear removes every entry and forgets every ghost key.
func (s *store[K, V]) clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.entries.Clear()
	s.ghosts = make(map[K]*node[K, V])
	var empty queue[K, V]
	s.small, s.main, s.ghost = empty, empty, empty
}

// len returns the number of entries.
func (s *store[K, V]) len() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.small.len + s.main.len
}

// all yields every entry in no particular order without blocking writers.
func (s *store[K, V]) all() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		s.entries.Range(func(_, v any) bool {
			n := v.(*node[K, V])
			return yield(n.key, n.value)
		})
	}
}

// evict removes exactly one entry, moving entries between queues until it
// finds one to drop. The caller must hold mutex.
func (s *store[K, V]) evict() {
	reinsertions := 0
	for {
		// Evict from small while it holds its share, so new keys cannot
		// grow into main's space. When main is empty, small is the only
		// place left to evict from.
		if s.small.len >= s.capacitySmall || s.main.len == 0 {
			n := s.small.pop()
			// More than one read while on probation: promote it. The
			// counter restarts so the key must earn its place in main again.
			if n.frequency.Load() > 1 {
				n.frequency.Store(0)
				s.main.push(n)
				continue
			}
			s.entries.Delete(n.key)
			// Remember the key, so a quick return goes straight to main.
			// The paper sizes ghost like main's share: it only needs to
			// remember keys long enough to notice the ones that return
			// within one pass through the cache.
			g := new(node[K, V])
			g.key = n.key
			s.ghosts[n.key] = g
			s.ghost.push(g)
			if s.ghost.len > s.capacity-s.capacitySmall {
				delete(s.ghosts, s.ghost.pop().key)
			}
			return
		}
		n := s.main.pop()
		// A read key gets another round with one fewer credit, so a key that
		// stops being read is evicted after at most frequencyMax rounds. If
		// every key in main is hot, the bound stops one eviction from walking
		// the whole queue.
		if f := n.frequency.Load(); f > 0 && reinsertions < reinsertionsMax {
			n.frequency.Store(f - 1)
			s.main.push(n)
			reinsertions++
			continue
		}
		s.entries.Delete(n.key)
		return
	}
}
