package singleflight

import "golang.org/x/sync/singleflight"

// Group is a type-safe wrapper around singleflight.Group.
// It deduplicates concurrent calls with the same key so only one
// executes while others wait and share the result.
type Group[T any] struct {
	g singleflight.Group
}

// Do executes fn once for a given key. If a duplicate call comes in
// while the first is still running, the duplicate caller waits and
// receives the same result.
func (g *Group[T]) Do(key string, fn func() (T, error)) (T, error) {
	v, err, _ := g.g.Do(key, func() (any, error) {
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
