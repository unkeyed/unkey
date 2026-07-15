// Package db provides MySQL access for the ratelimit service's cross-region
// count propagation. It is intentionally narrow: only the queries needed to
// upsert, import, and clean up rows in ratelimit_global_counters live here.
//
// Each region periodically writes its local count for active sliding-window
// cells. Other regions poll the table, sum foreign-region rows, and fold that
// value into local sliding-window math.
//
// # Code generation
//
// Run `make generate` from the repo root to regenerate query code. The
// pipeline (see generate.go and sqlc.json) clears stale *_generated.go files,
// builds the shared sqlc bulk-insert plugin, runs sqlc against the schema in
// pkg/mysql/schema and queries in queries/, and removes the scaffold db file
// that sqlc emits because database.go already provides a primary/replica-aware
// [Database] with [RW] and [RO] accessors.
package db
