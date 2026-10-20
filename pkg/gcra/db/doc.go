// Package db contains the generated queries of the ratelimit_gcra table.
// Callers supply the primary for Persist and cleanup, and a replica for
// ReadPeers. Connections remain owned by the caller.
//
// Both value columns of a row only grow, so Persist keeps the larger of the
// stored and the sent value and is safe to retry or reorder.
package db
