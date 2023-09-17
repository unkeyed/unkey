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

func NewNoopCache[T any]() Cache[T] {
	return &noopCache[T]{}
}
