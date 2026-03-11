// Package db contains sqlc-generated query code for ctrl/api.
//
// This package intentionally uses a writable-only connection because ctrl/api
// control-plane operations can require both reads and writes in the same flow.
package db
