// Package db provides MySQL access for the ratelimit service's cross-region
// denial propagation. It is intentionally narrow: only the queries needed to
// upsert, list, and clean up rows in ratelimit_blocklist live here.
//
// The service writes one row per (workspace, namespace, identifier, duration)
// when a node sees an identifier breach a limit for the first time in a
// window. Every region polls the table on a fixed interval and uses each row
// to inflate the local sliding-window counter so the same identifier is also
// denied in regions that have not yet seen the abuse traffic firsthand.
//
// # Code Generation
//
// Run `make generate` from the repo root to regenerate query code. The
// pipeline (see generate.go and sqlc.json) clears stale *_generated.go files,
// runs sqlc against the schema in pkg/mysql/schema and queries in queries/,
// and removes the scaffold db file that sqlc emits because database.go
// already provides a primary/replica-aware [Database] with [RW] and [RO]
// accessors.
package db
