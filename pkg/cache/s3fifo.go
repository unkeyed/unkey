package cache

// This file implements S3-FIFO, the policy the cache uses to decide which
// entry to drop when it is full. The explanation below assumes that you know
// a map-based cache with LRU eviction.
//
// An LRU cache drops the entry that was used least recently. To track this,
// every read moves its entry to the front of a shared list. Every read
// therefore writes shared state and needs a lock. LRU also handles a common
// pattern badly: in real traffic, most keys are read once and never again.
// These keys are called one-hit wonders. Each of them looks recently used, so
// a burst of them pushes out the keys that are actually in demand.
//
// S3-FIFO never moves an entry when it is read. A read only increments a
// small counter on the entry, up to `frequencyMax`. The cache looks at these
// counters later, when a write needs room. It keeps entries in three
// first-in-first-out (FIFO) queues. New entries go in at the head of a queue,
// and the oldest entries leave at the tail:
//
//	small  about 10% of capacity. Every new key starts here.
//	main   the rest of capacity. Keys that were read again while in `small`.
//	ghost  keys only, no values. Keys that `small` recently dropped.
//
// `small` and `main` together hold at most `capacity` entries. `ghost` is
// sized like `main`, but it stores only keys, so it uses little memory.
//
// When the cache is full and a new key arrives, `evict` removes one entry.
// While `small` holds at least its 10% share, or when `main` is empty,
// `evict` takes the oldest entry of `small` and checks how often it was read:
//
//	more than once  -> it is not a one-hit wonder. Move it to `main`, reset
//	                   its counter, and continue with the next entry.
//	at most once    -> drop it and remember its key in `ghost`.
//
// Otherwise, `evict` takes the oldest entry of `main`:
//
//	counter above 0 -> it was read since it was last checked. Put it back at
//	                   the head of `main`, decrement its counter, and
//	                   continue with the next entry.
//	counter at 0    -> drop it.
//
// The `ghost` queue corrects mistakes. If a key comes back while `ghost` still
// remembers it, `small` dropped it too early. In that case, `set` puts the key
// directly into `main`.
//
// Here is an example with a `capacity` of 10, so the share of `small` is 1.
// Keys k0 to k9 fill `small`, and k0 is read twice. Writing a new key then
// does this:
//
//	small (oldest first): k0 k1 ... k9    main: -
//	k0 was read twice  -> move k0 to `main`
//	k1 was never read  -> drop k1, remember k1 in `ghost`
//	small: k2 ... k9 new                  main: k0
//
// A plain FIFO cache would have dropped k0, the only key in use.
//
// This design works well under concurrency. Reads never change the queues,
// so a cache hit writes at most one atomic counter, and a key that is already
// at `frequencyMax` writes nothing. Entries are stored in a `sync.Map`, so
// lookups take no lock. Only `set`, `delete`, and `clear` change the queues,
// and they hold `mutex` while they do.
//
// The design also works well in simulation. Eviction runs inside the `set`
// call that needs room, on the caller's goroutine, and never uses randomness.
// The same sequence of calls therefore always evicts the same keys.

import (
	"iter"
	"sync"
	"sync/atomic"

	"github.com/unkeyed/unkey/pkg/ringbuffer"
)

const (
	// frequencyMax caps the read counter, so a key that was hot once loses its
	// protection after a few trips through the `main` queue without reads.
	frequencyMax = 3
	// reinsertionsMax bounds how many `main` entries one eviction may
	// give a second chance before it evicts the next one.
	reinsertionsMax = 20
)

// node is one entry in a queue. Only `frequency` changes while other
// goroutines can read the node; `key` and `value` are fixed once the node is
// published.
type node[K comparable, V any] struct {
	key       K
	value     V
	frequency atomic.Int32
	queue     *queue[K, V]
	prev      *node[K, V]
	next      *node[K, V]
}

// queue is a doubly linked FIFO. `push` adds at `head` and `pop` removes from
// `tail`, so `tail` is always the oldest entry.
type queue[K comparable, V any] struct {
	head *node[K, V]
	tail *node[K, V]
	len  int
}

func (q *queue[K, V]) push(n *node[K, V]) {
	n.queue = q
	n.prev = nil //faultline:equivalent every pushed node is new or was unlinked, so `prev` is already nil
	n.next = q.head
	if q.head != nil {
		q.head.prev = n
	} else {
		q.tail = n
	}
	q.head = n
	q.len++
}

// remove unlinks `n` from anywhere in `q`.
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
	n.prev, n.next, n.queue = nil, nil, nil //faultline:equivalent unlinked nodes are relinked by `push` or `replace` before any read; clearing only helps the GC
	q.len--
}

// pop removes and returns the oldest entry. `q` must not be empty.
func (q *queue[K, V]) pop() *node[K, V] {
	n := q.tail
	q.remove(n)
	return n
}

// replace puts `n` at the position of `old` in `q`.
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
	old.prev, old.next, old.queue = nil, nil, nil //faultline:equivalent unlinked nodes are relinked by `push` or `replace` before any read; clearing only helps the GC
}

// store is a fixed-capacity map that evicts with S3-FIFO
// (https://s3fifo.com). See the comment at the top of this file for how the
// policy works.
//
// `entries` holds every live node and is the only field reads touch. The other
// fields are the policy's bookkeeping and change only under `mutex`.
//
// The `ghost` queue is two fields. `ghostRing` holds the last `capacityGhost`
// keys that `small` dropped, in order, and forgets the oldest one first.
// `ghostIndex` maps each remembered key to the push that added it, so `set`
// can ask "was this key evicted recently?" in O(1). `ghostPushes` counts every
// push. A key that returns is deleted from `ghostIndex` but stays in
// `ghostRing` until the ring overwrites it.
type store[K comparable, V any] struct {
	entries sync.Map

	mutex       sync.Mutex
	small       queue[K, V]
	main        queue[K, V]
	ghostRing   *ringbuffer.Ring[K]
	ghostIndex  map[K]uint64
	ghostPushes uint64
	// capacity bounds `small.len + main.len`, the number of live entries. The
	// `ghost` queue holds no values and does not count toward it.
	capacity int
	// capacitySmall is the share of `capacity` for `small`. It is a target,
	// not a hard limit: new keys always enter `small`, so `small` can grow past
	// it while `main` is short of its share. `evict` brings it back by evicting
	// from `small` first whenever `small` holds `capacitySmall` or more. `main`
	// has no field of its own; it gets whatever capacity `small` does not use.
	capacitySmall int
	// capacityGhost is the capacity of `ghostRing`. The paper sizes it
	// like the share of `main`: it only needs to remember keys long enough to
	// notice the ones that return within one pass through the cache.
	capacityGhost int
}

// newStore returns an empty store that holds at most `capacity` entries.
func newStore[K comparable, V any](capacity int) *store[K, V] {
	s := new(store[K, V])
	s.capacity = capacity
	s.capacitySmall = max(1, capacity/10)
	s.capacityGhost = capacity - s.capacitySmall
	s.ghostRing = ringbuffer.New[K](s.capacityGhost)
	s.ghostIndex = make(map[K]uint64)
	return s
}

// get returns the value for `key` and counts the read toward its `frequency`.
func (s *store[K, V]) get(key K) (V, bool) {
	v, ok := s.entries.Load(key)
	if !ok {
		var zero V
		return zero, false
	}
	n := v.(*node[K, V])
	// Load before CompareAndSwap so a hot key at `frequencyMax` causes no
	// write. A lost race only drops one count, which the policy tolerates.
	if f := n.frequency.Load(); f < frequencyMax {
		n.frequency.CompareAndSwap(f, f+1)
	}
	return n.value, true
}

// set writes `key` and reports whether it replaced a value or evicted another
// entry to make room. Both answers fall out of work `set` does anyway.
func (s *store[K, V]) set(key K, value V) (replaced, evicted bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	n := new(node[K, V])
	n.key, n.value = key, value
	// An update builds a new node instead of writing `old.value`, because
	// readers may be reading `old` right now without a lock. The new node
	// takes the place of `old` in its queue and keeps its counter, so an
	// update neither resets the key's eviction order nor counts as a read.
	if v, ok := s.entries.Load(key); ok {
		old := v.(*node[K, V])
		n.frequency.Store(old.frequency.Load())
		old.queue.replace(old, n)
		s.entries.Store(key, n)
		return true, false
	}
	// A key that `small` evicted recently has proven it comes back, so it
	// skips probation and goes straight to `main`. Check before `evict`,
	// which can overwrite the ghost of this key.
	_, returning := s.ghostIndex[key]
	delete(s.ghostIndex, key)
	// Live entries never exceed `capacity` and an insert adds one, so one
	// eviction always makes room.
	if s.small.len+s.main.len >= s.capacity {
		s.evict()
		evicted = true
	}
	if returning {
		s.main.push(n)
	} else {
		s.small.push(n)
	}
	s.entries.Store(key, n)
	return false, evicted
}

// delete removes `key` and reports whether it was present.
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

// deleteIf removes `key` if `drop` returns true for its value, and reports
// whether it removed it. `drop` runs under `mutex`, so no write can land
// between the check and the removal.
func (s *store[K, V]) deleteIf(key K, drop func(V) bool) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	v, ok := s.entries.Load(key)
	if !ok {
		return false
	}
	n := v.(*node[K, V])
	if !drop(n.value) {
		return false
	}
	s.entries.Delete(key)
	n.queue.remove(n)
	return true
}

// clear removes every entry and forgets every `ghost` key.
func (s *store[K, V]) clear() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.entries.Clear()
	s.ghostRing.Clear()
	s.ghostIndex = make(map[K]uint64)
	var empty queue[K, V]
	s.small, s.main = empty, empty
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
// finds one to drop. The caller must hold `mutex`.
func (s *store[K, V]) evict() {
	reinsertions := 0
	for {
		// Evict from `small` while it holds its share, so new keys cannot
		// grow into the space of `main`. When `main` is empty, `small` is the
		// only place left to evict from.
		if s.small.len >= s.capacitySmall || s.main.len == 0 {
			n := s.small.pop()
			// More than one read while on probation: promote it. The
			// counter restarts so the key must earn its place in `main` again.
			if n.frequency.Load() > 1 {
				n.frequency.Store(0)
				s.main.push(n)
				continue
			}
			s.entries.Delete(n.key)
			s.remember(n.key)
			return
		}
		n := s.main.pop()
		// A read key gets another round with one fewer credit, so a key that
		// stops being read is evicted after at most `frequencyMax` rounds. If
		// every key in `main` is hot, the bound stops one eviction from walking
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

// remember adds `key` to the `ghost` queue, so a quick return goes straight
// to `main`. The caller must hold `mutex`.
func (s *store[K, V]) remember(key K) {
	push := s.ghostPushes
	s.ghostPushes++
	s.ghostIndex[key] = push
	oldest, full := s.ghostRing.Push(key)
	// The ring holds `capacityGhost` keys, so the key it overwrites came from
	// that many pushes ago. If that key returned and was dropped again, a
	// newer push remembers it, and it must stay.
	if full && s.ghostIndex[oldest] == push-uint64(s.capacityGhost) {
		delete(s.ghostIndex, oldest)
	}
}
