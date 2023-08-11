package server

import (
	"context"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
)

// withCache builds a pullthrough cache function to wrap a database call.
// Example:
// api, found, err := withCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
func withCache[T any](c cache.Cache[T], loadFromDatabase func(ctx context.Context, identifier string) (T, bool, error)) func(ctx context.Context, identifier string) (T, bool, error) {
	return func(ctx context.Context, identifier string) (value T, found bool, err error) {
		value, found = c.Get(ctx, identifier)
		if !found {
			value, found, err = loadFromDatabase(ctx, identifier)
			if err != nil {
				return
			}
			if found {
				c.Set(ctx, identifier, value)
			}
		}
		return
	}
}
