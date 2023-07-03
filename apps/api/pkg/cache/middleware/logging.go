package middleware

import (
	"context"

	"github.com/chronark/unkey/apps/api/pkg/cache"
	"go.uber.org/zap"
)

type loggingMiddleware[T any] struct {
	next   cache.Cache[T]
	logger *zap.Logger
}

func WithLogging[T any](c cache.Cache[T], l *zap.Logger) cache.Cache[T] {
	return &loggingMiddleware[T]{next: c, logger: l}
}

func (mw *loggingMiddleware[T]) Get(ctx context.Context, key string) (T, bool) {

	value, hit := mw.next.Get(ctx, key)
	mw.logger.Info("cache.get", zap.String("key", key), zap.Bool("hit", hit))
	return value, hit
}
func (mw *loggingMiddleware[T]) Set(ctx context.Context, key string, value T) {
	mw.logger.Info("cache.set", zap.String("key", key))

	mw.next.Set(ctx, key, value)

}
func (mw *loggingMiddleware[T]) Remove(ctx context.Context, key string) {
	mw.logger.Info("cache.remove", zap.String("key", key))

	mw.next.Remove(ctx, key)

}
