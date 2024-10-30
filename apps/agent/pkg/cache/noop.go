package cache

import (
	"context"
)

type noopCache[T any] struct{}

func (c *noopCache[T]) Get(ctx context.Context, key string) (value T, hit CacheHit) {
	var t T
	return t, Miss
}
func (c *noopCache[T]) Set(ctx context.Context, key string, value T) {}
func (c *noopCache[T]) SetNull(ctx context.Context, key string)      {}

func (c *noopCache[T]) Remove(ctx context.Context, key string) {}

func (c *noopCache[T]) Dump(ctx context.Context) ([]byte, error) {
	return []byte{}, nil
}
func (c *noopCache[T]) Restore(ctx context.Context, data []byte) error {
	return nil
}
func (c *noopCache[T]) Clear(ctx context.Context) {}

func NewNoopCache[T any]() Cache[T] {
	return &noopCache[T]{}
}
