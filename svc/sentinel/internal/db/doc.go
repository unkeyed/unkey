// Package db provides read-only database access for the sentinel proxy.
//
// Sentinel routes incoming requests to deployment instances by looking up
// deployments, regions, and instances. All queries are read-only because
// sentinel never mutates routing state; the control plane owns those writes.
//
// The package uses sqlc to generate type-safe query methods from SQL files in
// the queries/ subdirectory. Hand-written code in database.go provides the
// [New] constructor that connects to a MySQL read replica with tracing and
// metrics.
//
// # Key Types
//
// [Querier] is the generated interface listing all available queries. [New]
// returns a [Querier] backed by a read replica connection. Callers use the
// returned close function to release the underlying connection pool.
//
// # Code Generation
//
// Run `go generate ./...` from this directory to regenerate query code. The
// generate pipeline clears old *_generated.go files, runs sqlc, and removes
// the temporary db scaffold file that sqlc emits (see generate.go and
// sqlc.json for details).
package db
