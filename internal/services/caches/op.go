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

// WorkspaceByOrgIDOp is the cache write policy for organization to workspace
// lookups. A missing row is not stored. Caching that miss hides a workspace
// that is committed immediately afterwards for the fresh window, and for the
// stale window after that.
func WorkspaceByOrgIDOp(err error) cache.Op {
	if db.IsNotFound(err) {
		return cache.Noop
	}
	return DefaultFindFirstOp(err)
}
