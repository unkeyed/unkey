package cache

import (
	"context"
)

// withCache builds a pullthrough cache function to wrap a database call.
// Example:
// api, found, err := withCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
func WithCache[T any](c Cache[T], loadFromDatabase func(ctx context.Context, identifier string) (T, bool, error)) func(ctx context.Context, identifier string) (T, bool, error) {
	return func(ctx context.Context, identifier string) (T, bool, error) {
		value, hit := c.Get(ctx, identifier)

		if hit == Hit {
			return value, true, nil
		}
		if hit == Null {
			return value, false, nil
		}

		value, found, err := loadFromDatabase(ctx, identifier)
		if err != nil {
			return value, false, err
		}
		if found {
			c.Set(ctx, identifier, value)
			return value, true, nil
		} else {
			c.SetNull(ctx, identifier)
			return value, false, nil
		}
	}
}
