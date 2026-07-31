package singleflight

import (
	"strconv"
	"sync"

	"golang.org/x/sync/singleflight"
)

// Group is a type-safe wrapper around singleflight.Group.
// It deduplicates concurrent calls with the same key so only one
// executes while others wait and share the result.
type Group[K comparable, T any] struct {
	g singleflight.Group

	keysMu sync.Mutex
	keys   map[K]string
	nextID uint64
}

// Do executes fn once for a given key. If a duplicate call comes in
// while the first is still running, the duplicate caller waits and
// receives the same result.
func (g *Group[K, T]) Do(key K, fn func() (T, error)) (T, error) {
	flightKey := g.acquireKey(key)
	defer g.releaseKey(key, flightKey)

	v, err, _ := g.g.Do(flightKey, func() (any, error) {
		return fn()
	})
	if err != nil {
		var zero T
		return zero, err
	}
	if v == nil {
		var zero T
		return zero, nil
	}
	return v.(T), nil
}

// x/sync/singleflight only accepts string keys. Assigning IDs under a typed
// map preserves equality for any comparable key without unsafe formatting or
// hashing that could coalesce two different keys.
func (g *Group[K, T]) acquireKey(key K) string {
	g.keysMu.Lock()
	defer g.keysMu.Unlock()

	if flightKey, ok := g.keys[key]; ok {
		return flightKey
	}

	if g.keys == nil {
		g.keys = make(map[K]string)
	}
	g.nextID++
	flightKey := strconv.FormatUint(g.nextID, 10)
	g.keys[key] = flightKey
	return flightKey
}

func (g *Group[K, T]) releaseKey(key K, flightKey string) {
	g.keysMu.Lock()
	defer g.keysMu.Unlock()

	if g.keys[key] == flightKey {
		delete(g.keys, key)
	}
}
