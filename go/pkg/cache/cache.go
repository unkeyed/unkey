package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/maypok86/otter"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/logging"
	"github.com/unkeyed/unkey/go/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type cache[T any] struct {
	otter             otter.Cache[string, swrEntry[T]]
	fresh             time.Duration
	stale             time.Duration
	refreshFromOrigin func(ctx context.Context, identifier string) (data T, ok bool)
	// If a key is stale, its identifier will be put into this channel and a goroutine refreshes it in the background
	refreshC chan string
	logger   logging.Logger
	resource string
	clock    clock.Clock
}

type Config[T any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// A handler that will be called to refetch data from the origin when necessary
	RefreshFromOrigin func(ctx context.Context, identifier string) (data T, ok bool)

	Logger logging.Logger

	// Start evicting the least recently used entry when the cache grows to MaxSize
	MaxSize int

	Resource string

	Clock clock.Clock
}

func New[T any](config Config[T]) (*cache[T], error) {

	builder, err := otter.NewBuilder[string, swrEntry[T]](config.MaxSize)
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("failed to create otter builder", ""))
	}

	otter, err := builder.CollectStats().Cost(func(key string, value swrEntry[T]) uint32 {
		return 1
	}).WithTTL(time.Hour).Build()
	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("failed to create otter cache", ""))
	}

	c := &cache[T]{
		otter:             otter,
		fresh:             config.Fresh,
		stale:             config.Stale,
		refreshFromOrigin: config.RefreshFromOrigin,
		refreshC:          make(chan string, 1000),
		logger:            config.Logger,
		resource:          config.Resource,
		clock:             config.Clock,
	}

	go c.runRefreshing()

	return c, nil

}

func (c cache[T]) Get(ctx context.Context, key string) (value T, hit CacheHit) {

	e, ok := c.otter.Get(key)
	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, Miss
	}

	now := c.clock.Now()

	if now.Before(e.Fresh) {

		return e.Value, e.Hit

	}
	if now.Before(e.Stale) {
		c.refreshC <- key

		return e.Value, e.Hit
	}

	c.otter.Delete(key)

	var t T
	return t, Miss

}

func (c cache[T]) SetNull(ctx context.Context, key string) {
	c.set(ctx, key)
}

func (c cache[T]) Set(ctx context.Context, key string, value T) {
	c.set(ctx, key, value)
}
func (c cache[T]) set(_ context.Context, key string, value ...T) {
	now := c.clock.Now()

	e := swrEntry[T]{
		Value: value[0],
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
		Hit:   Null,
	}
	if len(value) > 0 {
		e.Value = value[0]
		e.Hit = Hit
	} else {
		e.Hit = Miss
	}
	c.otter.Set(key, e)

}

func (c cache[T]) Remove(ctx context.Context, key string) {

	c.otter.Delete(key)

}

func (c cache[T]) Dump(ctx context.Context) ([]byte, error) {
	data := make(map[string]swrEntry[T])

	c.otter.Range(func(key string, entry swrEntry[T]) bool {
		data[key] = entry
		return true
	})

	b, err := json.Marshal(data)

	if err != nil {
		return nil, fault.Wrap(err, fault.WithDesc("failed to marshal cache data", ""))
	}
	return b, nil

}

func (c cache[T]) Restore(ctx context.Context, b []byte) error {

	data := make(map[string]swrEntry[T])
	err := json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	now := c.clock.Now()
	for key, entry := range data {
		if now.Before(entry.Fresh) {
			c.Set(ctx, key, entry.Value)
		} else if now.Before(entry.Stale) {
			c.refreshC <- key
		}
		// If the entry is older than, we don't restore it
	}
	return nil
}

func (c cache[T]) Clear(ctx context.Context) {
	c.otter.Clear()
}

func (c cache[T]) runRefreshing() {
	for {
		ctx := context.Background()
		identifier := <-c.refreshC

		ctx, span := tracing.Start(ctx, tracing.NewSpanName(fmt.Sprintf("cache.%s", c.resource), "refresh"))
		span.SetAttributes(attribute.String("identifier", identifier))
		t, ok := c.refreshFromOrigin(ctx, identifier)
		if !ok {
			span.AddEvent("identifier not found in origin")
			c.logger.Warn(ctx, "origin couldn't find data", slog.String("identifier", identifier))
			span.End()
			continue
		}
		c.Set(ctx, identifier, t)
		span.End()
	}

}

func (c cache[T]) SWR(ctx context.Context, identifier string) (T, bool) {

	value, hit := c.Get(ctx, identifier)

	if hit == Hit {
		return value, true
	}
	if hit == Null {
		return value, false
	}

	value, found := c.refreshFromOrigin(ctx, identifier)
	if found {
		c.Set(ctx, identifier, value)
		return value, true
	}
	c.SetNull(ctx, identifier)
	return value, false

}
