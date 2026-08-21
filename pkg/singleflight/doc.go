// Package singleflight deduplicates concurrent function calls by key while
// allowing calls for different keys to proceed independently.
//
// Waiting callers can cancel independently without canceling the executing
// caller. If that caller is canceled or panics, waiters retry with their own
// function and context. Groups must be constructed with [New].
package singleflight
