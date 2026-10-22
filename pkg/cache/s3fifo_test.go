package cache

import (
	"math/rand/v2"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireKeys[V any](t *testing.T, s *store[string, V], present []string, absent []string) {
	t.Helper()
	for _, k := range present {
		_, ok := s.get(k)
		require.True(t, ok, "%s must be cached", k)
	}
	for _, k := range absent {
		_, ok := s.get(k)
		require.False(t, ok, "%s must be evicted", k)
	}
}

// A plain FIFO evicts the oldest key. S3-FIFO first promotes an old key that
// was read more than once, so the next one-hit key goes instead.
func TestStoreSecondChanceInSmallQueue(t *testing.T) {
	s := newStore[string, int](10)
	keys := []string{"k0", "k1", "k2", "k3", "k4", "k5", "k6", "k7", "k8", "k9"}
	for i, k := range keys {
		replaced, evicted := s.set(k, i)
		require.False(t, replaced)
		require.False(t, evicted)
	}
	s.get("k0")
	s.get("k0")

	replaced, evicted := s.set("new", 10)
	require.False(t, replaced)
	require.True(t, evicted)
	require.Equal(t, 10, s.len())
	requireKeys(t, s, append([]string{"new", "k0"}, keys[2:]...), []string{"k1"})
}

// A key read only once stays in the small queue and is evicted like any
// one-hit key. The threshold is "more than one read", not "any read".
func TestStoreSingleReadIsNotPromoted(t *testing.T) {
	s := newStore[string, int](3)
	s.set("a", 1)
	s.set("b", 2)
	s.set("c", 3)
	s.get("a")

	s.set("d", 4)
	requireKeys(t, s, []string{"b", "c", "d"}, []string{"a"})
}

// A key that returns soon after eviction was evicted too early. Its key is
// still in the ghost queue, so it goes straight to the main queue, where a
// scan of one-hit keys cannot reach it.
func TestStoreGhostKeySkipsSmallQueue(t *testing.T) {
	s := newStore[string, int](10)
	for i := range 10 {
		s.set(string(rune('a'+i)), i)
	}
	s.set("x", 0)
	_, ok := s.get("a")
	require.False(t, ok)

	s.set("a", 1)
	for i := range 100 {
		s.set(string(rune(1000+i)), i)
	}
	requireKeys(t, s, []string{"a"}, []string{"b", "c", "x"})
}

// The main queue gives a read key another round and evicts the first key
// with no reads since its last round.
func TestStoreMainQueueReinsertsReadKeys(t *testing.T) {
	s := newStore[string, int](3)
	s.set("a", 1)
	s.set("b", 2)
	s.set("c", 3)
	for range 2 {
		s.get("a")
		s.get("b")
	}
	s.set("d", 4)
	require.True(t, s.delete("d"))
	s.set("c", 3)
	s.get("a")

	_, evicted := s.set("e", 5)
	require.True(t, evicted)
	requireKeys(t, s, []string{"a", "c", "e"}, []string{"b", "d"})
}

// Every key in the main queue was read, so without a bound the main queue
// would circle through all of them before it found a victim. After
// reinsertionsMax keys it evicts the next one, whatever its frequency.
func TestStoreBoundsMainQueueReinsertions(t *testing.T) {
	s := newStore[int, int](30)
	for i := range 30 {
		s.set(i, i)
	}
	for i := range 27 {
		s.get(i)
		s.get(i)
	}
	s.set(100, 0)
	for i := range 27 {
		for range 3 {
			s.get(i)
		}
	}
	for _, k := range []int{28, 29, 100} {
		require.True(t, s.delete(k))
	}
	s.set(27, 0)
	s.set(101, 0)
	s.set(102, 0)

	_, evicted := s.set(103, 0)
	require.True(t, evicted)
	for i := range 28 {
		_, ok := s.get(i)
		require.Equal(t, i != 20, ok, "key %d", i)
	}
}

func TestStoreUpdateKeepsPosition(t *testing.T) {
	s := newStore[string, int](3)
	s.set("a", 1)
	s.set("b", 2)
	s.set("c", 3)

	replaced, evicted := s.set("a", 10)
	require.True(t, replaced)
	require.False(t, evicted)
	require.Equal(t, 3, s.len())

	s.set("d", 4)
	requireKeys(t, s, []string{"b", "c", "d"}, []string{"a"})
}

func TestStoreUpdateKeepsFrequency(t *testing.T) {
	s := newStore[string, int](3)
	s.set("a", 1)
	s.set("b", 2)
	s.set("c", 3)
	s.get("a")
	s.get("a")
	s.set("a", 10)

	s.set("d", 4)
	requireKeys(t, s, []string{"a", "c", "d"}, []string{"b"})
	v, _ := s.get("a")
	require.Equal(t, 10, v)
}

func TestStoreDeleteAndClear(t *testing.T) {
	s := newStore[string, int](2)
	require.False(t, s.delete("missing"))
	s.set("a", 1)
	s.set("b", 2)
	require.True(t, s.delete("a"))
	require.False(t, s.delete("a"))
	require.Equal(t, 1, s.len())

	_, evicted := s.set("c", 3)
	require.False(t, evicted, "a deleted key must free its slot")

	s.clear()
	require.Zero(t, s.len())
	requireKeys(t, s, nil, []string{"b", "c"})
	for range s.all() {
		t.Fatal("cleared store must be empty")
	}
	s.set("a", 1)
	s.set("b", 2)
	_, evicted = s.set("c", 3)
	require.True(t, evicted)
}

// refEntry and refStore are a second S3-FIFO, written from the paper's rules
// with slices instead of linked lists, so the store can be checked against an
// implementation that shares none of its code. Queues are oldest first.
type refEntry struct {
	key, value, frequency int
}

type refStore struct {
	capacity, capacitySmall int
	small, main             []*refEntry
	ghost                   []int
	entries                 map[int]*refEntry
}

func newRefStore(capacity int) *refStore {
	return &refStore{capacity: capacity, capacitySmall: max(1, capacity/10), small: nil, main: nil, ghost: nil, entries: map[int]*refEntry{}}
}

func (r *refStore) get(key int) (int, bool) {
	e, ok := r.entries[key]
	if !ok {
		return 0, false
	}
	e.frequency = min(e.frequency+1, 3)
	return e.value, true
}

func (r *refStore) set(key, value int) bool {
	if e, ok := r.entries[key]; ok {
		e.value = value
		return true
	}
	for len(r.entries) >= r.capacity {
		r.evict()
	}
	e := &refEntry{key: key, value: value, frequency: 0}
	if i := slices.Index(r.ghost, key); i >= 0 {
		r.ghost = slices.Delete(r.ghost, i, i+1)
		r.main = append(r.main, e)
	} else {
		r.small = append(r.small, e)
	}
	r.entries[key] = e
	return false
}

func (r *refStore) evict() {
	for reinserted := 0; ; {
		if len(r.small) >= r.capacitySmall || len(r.main) == 0 {
			e := r.small[0]
			r.small = r.small[1:]
			if e.frequency > 1 {
				e.frequency = 0
				r.main = append(r.main, e)
				continue
			}
			delete(r.entries, e.key)
			r.ghost = append(r.ghost, e.key)
			if len(r.ghost) > r.capacity-r.capacitySmall {
				r.ghost = r.ghost[1:]
			}
			return
		}
		e := r.main[0]
		r.main = r.main[1:]
		if e.frequency > 0 && reinserted < 20 {
			e.frequency--
			r.main = append(r.main, e)
			reinserted++
			continue
		}
		delete(r.entries, e.key)
		return
	}
}

func (r *refStore) delete(key int) bool {
	e, ok := r.entries[key]
	if !ok {
		return false
	}
	delete(r.entries, key)
	unlink := func(q []*refEntry) []*refEntry {
		return slices.DeleteFunc(q, func(x *refEntry) bool { return x == e })
	}
	r.small, r.main = unlink(r.small), unlink(r.main)
	return true
}

func (r *refStore) clear() {
	*r = *newRefStore(r.capacity)
}

// TestStoreMatchesReference runs seeded random operations against the store
// and the reference. Every read must agree, including which keys were evicted.
// Half of the keys come from a small hot set, so entries build up frequency,
// reach the main queue, and decay there.
func TestStoreMatchesReference(t *testing.T) {
	for _, capacity := range []int{1, 2, 7, 64, 100} {
		for _, keySpace := range []int{capacity + capacity/4 + 1, capacity * 4} {
			testStoreMatchesReference(t, capacity, keySpace)
		}
	}
}

func testStoreMatchesReference(t *testing.T, capacity, keySpace int) {
	t.Helper()
	{
		rng := rand.New(rand.NewPCG(uint64(capacity), uint64(keySpace)))
		s := newStore[int, int](capacity)
		ref := newRefStore(capacity)
		hot := max(1, capacity/2)
		for i := range 50_000 {
			key := rng.IntN(keySpace)
			if rng.IntN(2) == 0 {
				key = rng.IntN(hot)
			}
			switch op := rng.IntN(10_000); {
			case op == 0:
				s.clear()
				ref.clear()
			case op < 6_000:
				v, ok := s.get(key)
				wantV, wantOK := ref.get(key)
				require.Equal(t, wantOK, ok, "op %d: get(%d)", i, key)
				require.Equal(t, wantV, v, "op %d: get(%d)", i, key)
			case op < 9_000:
				replaced, _ := s.set(key, i)
				require.Equal(t, ref.set(key, i), replaced, "op %d: set(%d)", i, key)
			default:
				require.Equal(t, ref.delete(key), s.delete(key), "op %d: delete(%d)", i, key)
			}
			require.Equal(t, len(ref.entries), s.len(), "op %d", i)
		}
		for k, v := range s.all() {
			require.Equal(t, ref.entries[k].value, v)
		}
	}
}

func TestStoreConcurrentUse(t *testing.T) {
	s := newStore[int, int](50)
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Go(func() {
			for i := range 20_000 {
				k := (i * (g + 1)) % 500
				if _, ok := s.get(k); !ok {
					s.set(k, i)
				}
				if i%97 == 0 {
					s.delete(k)
				}
			}
		})
	}
	wg.Wait()
	n := 0
	for range s.all() {
		n++
	}
	require.Equal(t, s.len(), n)
	require.LessOrEqual(t, n, 50)
}
