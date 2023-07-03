package cache

import (
	"context"
	"go.uber.org/zap"
	"sync"
	"time"
)

type swrEntry[T any] struct {
	Value T
	Fresh time.Time
	Stale time.Time
}

type cache[T any] struct {
	sync.RWMutex
	data              map[string]swrEntry[T]
	fresh             time.Duration
	stale             time.Duration
	refreshFromOrigin func(ctx context.Context, identifier string) (T, error)

	// If a key is stale, its identifier will be put into this channel and a goroutine refreshes it in the background
	refreshC chan string

	logger *zap.Logger
}

type Config[T any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// A handler that will be called to refetch data from the origin when necessary
	RefreshFromOrigin func(ctx context.Context, identifier string) (T, error)

	Logger *zap.Logger
}

func New[T any](config Config[T]) Cache[T] {

	c := &cache[T]{
		data:              make(map[string]swrEntry[T]),
		fresh:             config.Fresh,
		stale:             config.Stale,
		refreshFromOrigin: config.RefreshFromOrigin,
		refreshC:          make(chan string),
		logger:            config.Logger.With(zap.String("pkg", "cache")),
	}

	go c.runEviction()
	go c.runRefreshing()
	return c
}

func (c *cache[T]) runEviction() {
	for range time.NewTicker(time.Minute).C {
		now := time.Now()
		c.Lock()
		for key, val := range c.data {
			if val.Stale.After(now) {
				delete(c.data, key)
			}
		}
		c.Unlock()
	}

}

func (c *cache[T]) runRefreshing() {
	for {
		select {
		case identifier := <-c.refreshC:
			ctx := context.Background()
			t, err := c.refreshFromOrigin(ctx, identifier)
			if err != nil {
				c.logger.Error("unable to refresh", zap.String("identifier", identifier), zap.Error(err))
				continue
			}
			c.Set(ctx, identifier, t)
		}
	}
}

func (c *cache[T]) Get(ctx context.Context, key string) (value T, found bool) {
	c.RLock()
	e, ok := c.data[key]
	c.RUnlock()
	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, false
	}

	now := time.Now()

	if now.Before(e.Fresh) {
		return e.Value, true
	}
	if now.Before(e.Stale) {
		c.refreshC <- key
		return e.Value, true
	}

	c.Lock()
	delete(c.data, key)
	c.Unlock()

	var t T
	return t, false

}

func (c *cache[T]) Set(ctx context.Context, key string, value T) {
	now := time.Now()
	c.Lock()
	defer c.Unlock()
	c.data[key] = swrEntry[T]{
		Value: value,
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
	}

}

func (c *cache[T]) Remove(ctx context.Context, key string) {

	c.Lock()
	defer c.Unlock()

	delete(c.data, key)

}
