package flags

import "github.com/unkeyed/unkey/pkg/cache"

// NewNoop returns a Service that never reads the database, so every flag
// reports its code default. Use it where no database is configured.
func NewNoop() *service {
	return &service{db: nil, cache: cache.NewNoopCache[string, map[string]Flag]()}
}
