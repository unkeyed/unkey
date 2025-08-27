package caches

import (
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
)

// DefaultFindFirstOp returns the appropriate cache operation based on the sql error
func DefaultFindFirstOp(err error) cache.Op {
	if err == nil {
		// everything went well and we have a row response
		return cache.WriteValue
	}

	if db.IsNotFound(err) {
		// the response is empty, we need to store that the row does not exist
		return cache.WriteNull
	}

	// this is a noop in the cache
	return cache.Noop
}
