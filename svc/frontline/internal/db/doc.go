// Package db provides read-only database access for the frontline reverse proxy.
//
// Frontline resolves incoming requests to deployment targets by looking up
// routes, certificates, and instance state. All queries here are read-only
// because frontline does not mutate routing or certificate state; the policy
// engine uses a separate connection pool from pkg/db when it needs to
// decrement key credits.
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
// Run `make generate` from the repo root to regenerate query code. The
// generate pipeline clears old *_generated.go files, runs sqlc, and removes
// the temporary db scaffold file that sqlc emits (see generate.go and
// sqlc.json for details).
package db
