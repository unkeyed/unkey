// Package db provides database access for the keys service.
//
// It supports separate read and write replicas so that read-heavy key
// verification queries go to the replica while migration writes go to
// the primary.
//
// The package uses sqlc to generate type-safe query methods from SQL files
// in the queries/ subdirectory. Hand-written code in database.go provides
// the [Database] type that wraps the two replicas.
//
// # Code Generation
//
// Run `go generate ./...` from this directory to regenerate query code.
package db
