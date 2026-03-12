// Package db contains sqlc-generated query code for ctrl/worker.
//
// This package intentionally uses a writable-only database connection for
// control-plane consistency in workflow steps that perform writes then reads.
package db
