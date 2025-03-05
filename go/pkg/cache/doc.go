// Package cache provides a generic, thread-safe caching system with support
// for time-based expiration, custom eviction policies, and observability.
//
// The cache implementation uses a combination of LRU (Least Recently Used)
// and TTL (Time To Live) strategies to manage memory efficiently. It supports
// stale-while-revalidate (SWR) behavior, allowing expired items to be served
// while being refreshed asynchronously in the background.
//
// Basic usage:
//
//	// Create a new cache with default settings
//	c := cache.New[string, User](cache.Config[string, User]{
//	    Fresh:    time.Minute,    // Items considered fresh for 1 minute
//	    Stale:    time.Hour,      // Items can be served stale for up to 1 hour
//	    MaxSize:  10000,          // Store up to 10,000 items
//	    Logger:   logger,
//	    Resource: "users",        // For metrics and logging
//	})
//
//	// Store an item
//	c.Set(ctx, "user:123", user)
//
//	// Retrieve an item
//	user, hit := c.Get(ctx, "user:123")
//	if hit == cache.Hit {
//	    // Use the cached user
//	} else {
//	    // Cache miss, fetch from database
//	}
//
//	// SWR pattern
//	user, err := c.SWR(ctx, "user:123",
//	    func(ctx context.Context) (User, error) {
//	        // This will only be called if the cache doesn't have a fresh value
//	        return fetchUserFromDatabase(ctx, "123")
//	    },
//	    func(err error) cache.CacheHit {
//	        // Translate errors to cache behavior
//	        if errors.Is(err, sql.ErrNoRows) {
//	            return cache.Null  // Mark as explicitly not found
//	        }
//	        return cache.Miss      // Mark as a transient error
//	    },
//	)
package cache
