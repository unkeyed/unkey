// Package db contains generated persistence queries for versioned counter
// snapshots. Callers supply primary connections for Persist, DeleteExpired,
// DeleteWriterCounters and the compaction queries, and a regional replica for
// Read. Connections remain owned by the caller.
//
// Names identify counters globally; callers include any namespace or tenant
// scope in the name.
//
// Rows are identified by name and writer id. Read sums every row of a name
// except the rows of two writer ids, which the writer holds in memory, and
// returns an error if the sum does not fit in int64. Persist is safe to retry
// while the row is retained. Deleting a row removes its revision history, so a
// caller must ensure the writer cannot persist it again.
//
// A row whose compaction_after has passed is final: no writer writes it anymore.
// Compaction merges such rows of one name, with the rows it created or moved
// earlier, into a new row whose writer id starts with "compacted_". The new row
// takes the lowest compaction_level of its sources. A row with nothing to merge
// with moves up a level and waits longer. ListCompactableNames reads without
// locks. LockCompactableRowsOfName, LockParkedRowsOfName, InsertCompactedRow,
// DeleteMergedRows and DeferCompaction belong to one transaction on the primary.
package db
