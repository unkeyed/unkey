package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	ottercache "github.com/maypok86/otter"
	"github.com/unkeyed/unkey/apps/agent/pkg/logging"
	"github.com/unkeyed/unkey/apps/agent/pkg/metrics"
	"github.com/unkeyed/unkey/apps/agent/pkg/repeat"
	"github.com/unkeyed/unkey/apps/agent/pkg/tracing"
	"go.opentelemetry.io/otel/attribute"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

type otter[T any] struct {
	cache             ottercache.Cache[string, swrEntry[T]]
	fresh             time.Duration
	stale             time.Duration
	refreshFromOrigin func(ctx context.Context, identifier string) (data T, ok bool)
	// If a key is stale, its identifier will be put into this channel and a goroutine refreshes it in the background
	refreshC chan string
	metrics  metrics.Metrics
	logger   logging.Logger
	resource string
}

type OtterConfig[T any] struct {
	// How long the data is considered fresh
	// Subsequent requests in this time will try to use the cache
	Fresh time.Duration

	// Subsequent requests that are not fresh but within the stale time will return cached data but also trigger
	// fetching from the origin server
	Stale time.Duration

	// A handler that will be called to refetch data from the origin when necessary
	RefreshFromOrigin func(ctx context.Context, identifier string) (data T, ok bool)

	Logger  logging.Logger
	Metrics metrics.Metrics

	// Start evicting the least recently used entry when the cache grows to MaxSize
	MaxSize int

	Resource string
}

func NewOtter[T any](config OtterConfig[T]) (Cache[T], error) {

	builder, err := ottercache.NewBuilder[string, swrEntry[T]](config.MaxSize)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to create otter builder"))
	}
	cache, err := builder.CollectStats().Cost(func(key string, value swrEntry[T]) uint32 {
		return 1
	}).WithTTL(time.Hour).Build()
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to create otter cache"))
	}

	o := &otter[T]{
		cache:             cache,
		fresh:             config.Fresh,
		stale:             config.Stale,
		refreshFromOrigin: config.RefreshFromOrigin,
		refreshC:          make(chan string, 1000),
		logger:            config.Logger,
		metrics:           config.Metrics,
		resource:          config.Resource,
	}

	go o.runRefreshing()

	repeat.Every(time.Minute, func() {
		o.metrics.Record(CacheStats{
			Resource: o.resource,
			Size:     o.cache.Size(),
			Rejected: o.cache.Stats().RejectedSets(),
		})
	})

	return o, nil

}

func (c *otter[T]) Get(ctx context.Context, key string) (value T, hit CacheHit) {

	e, ok := c.cache.Get(key)
	if !ok {
		// This hack is necessary because you can not return nil as T
		var t T
		return t, Miss
	}

	now := time.Now()

	if now.Before(e.Fresh) {

		return e.Value, e.Hit

	}
	if now.Before(e.Stale) {
		c.refreshC <- key

		return e.Value, e.Hit
	}

	c.cache.Delete(key)

	var t T
	return t, Miss

}

func (c *otter[T]) SetNull(ctx context.Context, key string) {
	c.set(ctx, key)
}

func (c *otter[T]) Set(ctx context.Context, key string, value T) {
	c.set(ctx, key, value)
}
func (c *otter[T]) set(ctx context.Context, key string, value ...T) {
	now := time.Now()

	e := swrEntry[T]{
		Value: value[0],
		Fresh: now.Add(c.fresh),
		Stale: now.Add(c.stale),
	}
	if len(value) > 0 {
		e.Value = value[0]
		e.Hit = Hit
	} else {
		e.Hit = Miss
	}
	c.cache.Set(key, e)

}

func (c *otter[T]) Remove(ctx context.Context, key string) {

	c.cache.Delete(key)

}

func (c *otter[T]) Dump(ctx context.Context) ([]byte, error) {
	data := make(map[string]swrEntry[T])

	c.cache.Range(func(key string, entry swrEntry[T]) bool {
		data[key] = entry
		return true
	})

	return json.Marshal(data)

}

func (c *otter[T]) Restore(ctx context.Context, b []byte) error {

	data := make(map[string]swrEntry[T])
	err := json.Unmarshal(b, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}
	now := time.Now()
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

func (c *otter[T]) Clear(ctx context.Context) {
	c.cache.Clear()
}

func (c *otter[T]) runRefreshing() {
	for {
		identifier := <-c.refreshC

		ctx, span := tracing.Start(context.Background(), tracing.NewSpanName(fmt.Sprintf("cache.%s", c.resource), "refresh"))
		span.SetAttributes(attribute.String("identifier", identifier))
		t, ok := c.refreshFromOrigin(ctx, identifier)
		if !ok {
			span.AddEvent("identifier not found in origin")
			c.logger.Warn().Str("identifier", identifier).Msg("origin couldn't find")
			span.End()
			continue
		}
		c.Set(ctx, identifier, t)
		span.End()
	}

}
