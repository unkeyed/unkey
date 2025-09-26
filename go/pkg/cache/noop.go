package cache

import (
	"context"
)

type noopCache[K comparable, V any] struct{}

func (c *noopCache[K, V]) Get(ctx context.Context, key K) (value V, hit CacheHit) {
	var v V
	return v, Miss
}
func (c *noopCache[K, V]) Set(ctx context.Context, key K, value V) {}
func (c *noopCache[K, V]) SetNull(ctx context.Context, key K)      {}

func (c *noopCache[K, V]) Remove(ctx context.Context, keys ...K) {}

func (c *noopCache[K, V]) Dump(ctx context.Context) ([]byte, error) {
	return []byte{}, nil
}
func (c *noopCache[K, V]) Restore(ctx context.Context, data []byte) error {
	return nil
}
func (c *noopCache[K, V]) Clear(ctx context.Context) {}

func (c *noopCache[K, V]) Name() string {
	return "noop"
}

func (c *noopCache[K, V]) SWR(ctx context.Context, key K, refreshFromOrigin func(context.Context) (V, error), op func(err error) Op) (V, CacheHit, error) {
	var v V
	return v, Miss, nil
}

func NewNoopCache[K comparable, V any]() Cache[K, V] {
	return &noopCache[K, V]{}
}
