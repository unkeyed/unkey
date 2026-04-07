// Package semaphore provides a bounded concurrency primitive.
//
// Use [New] to create a semaphore with a fixed number of slots, then call
// [Semaphore.Do] to run a function in a new goroutine while respecting the
// concurrency limit. Do blocks until a slot is available.
package semaphore
