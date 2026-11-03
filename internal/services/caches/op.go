package caches

import (
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
)

// DefaultFindFirstOp returns the appropriate cache operation based on the sql error
func DefaultFindFirstOp(err error) cache.Op {
	if db.IsNotFound(err) {
		// the response is empty, we need to store that the row does not exist
		return cache.WriteNull
	}

	if err == nil {
		// everything went well and we have a row response
		return cache.WriteValue
	}

	// this is a noop in the cache
	return cache.Noop
}

// FoundOnlyOp caches rows but never a miss, for lookups whose row can start
// matching moments after a miss and must be seen immediately.
func FoundOnlyOp(err error) cache.Op {
	if err == nil {
		return cache.WriteValue
	}
	return cache.Noop
}
