package caches

import (
	"database/sql"
	"errors"

	"github.com/unkeyed/unkey/go/pkg/cache"
)

// DefaultFindFirstOp returns the appropriate cache operation based on the SQL error.
// It helps the cache system determine how to handle different database query results.
//
// This function evaluates the error returned from a database query and decides:
// - If the error is nil (successful query), write the value to cache
// - If the error is sql.ErrNoRows (item not found), write a null marker to cache
// - For any other error, perform no caching operation
//
// This helper is used across multiple cache operations to consistently handle
// the common patterns of database lookups, especially for "find by ID" type queries.
//
// Parameters:
//   - err: The error returned from a database query
//
// Returns a cache.Op indicating what operation should be performed on the cache:
//   - cache.WriteValue: Write the result to cache
//   - cache.WriteNull: Write a null marker to cache
//   - cache.Noop: Don't modify the cache
func DefaultFindFirstOp(err error) cache.Op {
	if err == nil {
		// everything went well and we have a row response
		return cache.WriteValue
	}
	if errors.Is(err, sql.ErrNoRows) {
		// the response is empty, we need to store that the row does not exist
		return cache.WriteNull
	}
	// this is a noop in the cache
	return cache.Noop

}
