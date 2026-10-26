// Package db contains generated persistence queries for versioned counter
// snapshots. Callers supply primary connections for Persist and DeleteExpired,
// and a regional replica for Read. Connections remain owned by the caller.
//
// Names identify counters globally; callers include any namespace or tenant
// scope in the name.
//
// Rows are identified by name and writer id. Read sums every row of a name
// except the rows of two writer ids, which the writer holds in memory, and
// returns an error if the sum does not fit in int64. Persist is safe to retry
// while the row is retained. Deleting a row removes its revision history, so a
// caller must ensure the writer cannot persist it again.
package db
