// Package globalcounter combines local signed counters with cached peer totals.
//
// # Writer ids
//
// New generates a fresh process id for each Counter, so lost local state cannot
// reuse a previous writer's revisions. A writer id is the process id and a
// sequence, such as "x7Kq_3". The sequence starts at one and goes up about every
// 15 minutes, with 20% jitter. A writer keeps a separate signed contribution and
// revision for each name and writer id. A new writer id starts at zero and does
// not inherit the previous one; a name's total is the sum over all writer ids, so
// a rotation resets nothing.
//
// Add updates the entry of the current writer id and queues an absolute snapshot
// with an increasing revision. MySQL accepts only newer revisions of a row, so
// retries cannot double-count and deductions can decrease the stored
// contribution.
//
// # Reads
//
// Get adds this writer's contribution to the sum of every other row. A writer
// holds the current and the previous writer id in memory. Those come from memory,
// including changes not yet persisted, and the writer's own rows of those two ids
// are left out of the peer sum. Older writer ids are left out of its local total,
// so their rows count like any other writer's. The previous id stays in memory for a
// whole rotation after its last change, which is far longer than the replica lag
// and the cache refresh, so in normal operation a rotation does not make the
// total dip. If the final snapshot of an id was never persisted, or the replica
// is further behind than a rotation, the total can change when the id ages out.
// The peer sum is cached, with a key that includes the current sequence, so a
// rotation reloads instead of serving a sum that counts a writer id twice or not
// at all.
//
// Reads use a one-second fresh period and a five-second stale deadline, both
// measured from loading. Refreshes cannot reveal changes that other writers
// have not persisted or that the replica has not received. Unused names are not
// refreshed in the background.
//
// # Persisting
//
// Persistence uses batches of up to 10,000 snapshots and a one-second persist
// interval. Of several snapshots of the same row, name and writer id, only the
// newest revision is written; another writer id of the name is another row. A
// write circuit breaker wraps up to three attempts for transient failures, with a
// five-second deadline for the whole batch. Final failures and circuit
// rejections are logged, and the batch is dropped. A full 100,000-entry queue
// also drops new snapshots.
//
// Dropping a snapshot does not undo its local count while its writer id is held
// in memory. A later Add in the same sequence persists the cumulative value,
// including changes in dropped snapshots. A writer id stops receiving Adds when
// it becomes the previous one, so every eviction pass queues the entries of the
// previous id again until MySQL acknowledges their newest revision. An entry that
// is still unacknowledged when its id leaves memory is lost and counted as
// abandoned. Nothing writes an id again once it is older than the previous one.
// Successful Add calls are not durable acknowledgements. A process exit can lose
// pending changes.
//
// The database sets the compaction_after of a row to its own time plus 60 minutes
// whenever it accepts a newer revision. A writer stops writing a row when it
// forgets the writer id, at most about 36 minutes after the id started, so
// compaction_after is the margin for a write that is still in flight. A pod that
// stalls for longer than that is outside the model, and its late write would
// recreate a merged row and be counted twice.
//
// # Eviction and shutdown
//
// DeleteAfter is cleanup metadata, not a read TTL. A background eviction pass
// runs about once a minute, with 20% jitter. It first forgets the entries of
// writer ids older than the previous one, keeping their rows, and queues the
// previous id's unacknowledged entries again. Neither needs the database. It then
// deletes rows of every writer that passed their delete_after more than a minute
// ago, and forgets this writer's expired entries together with its own rows for
// them. It keeps those entries when a delete fails. An entry is forgotten only a
// minute after both its delete_after and its latest change, so a snapshot that is
// still in flight cannot arrive after the name has started again. A change after
// that starts this writer's count for the name from zero, and a change that races
// with the pass itself can lose its row, so nothing should change a counter after
// its delete_after. Close stops eviction, rotation and compaction, drains queued
// snapshots, then persists the entries of the current and the previous writer id,
// including dropped snapshots, within one deadline.
//
// # Compaction
//
// Every restart adds a process, and every rotation adds a writer id, so rows
// would pile up. A background pass, run by every Counter about every 10 minutes
// with 50% jitter, merges rows that nobody writes anymore. A row is compactable
// once its compaction_after has passed by the database clock. That is 60 minutes
// after its last accepted write, so every writer has forgotten its writer id and
// the row is final.
//
// For each name with a compactable row, the pass locks the compactable rows of
// that name, skipping rows another Counter holds, and the rows that compaction
// created or moved earlier, which no writer writes either. A ready row therefore
// makes the pass merge its name's older compacted rows in too. With two or more rows,
// the pass sums them into one new row with a random writer id, keeping the latest
// delete_after (no deletion time wins), and deletes the originals in the same
// transaction. The total of the name does not change.
//
// Every row has a compaction level. Level 0 is for a row that a writer wrote,
// which waits for its 60-minute margin. A merged row takes the lowest level of its
// sources, but at least 1, so a counter that still gets new rows stays at a low
// level. A row with nothing to merge with, or whose sum would overflow int64,
// stays as it is, and its level goes up by one. Its compaction_after then moves
// ahead by 10 minutes times two to the power of the new level, up to a year: 20
// minutes at level 1, 40 at level 2, and so on. A counter that nobody writes is
// looked at less and less often, and the ready rows of a counter that is written
// are merged by the next pass.
//
// A transaction handles up to 100 names and locks at most 500 rows, ready and
// parked rows together, and a pass handles at most 5,000 rows, so the work is
// bounded by rows and not by names, and Counters sharing a backlog take different
// rows. A transaction runs at READ COMMITTED, so it takes no gap locks and does
// not hold up a writer's persist for the same name. The transaction keeps no state
// outside the database, so repeating it after a lost reply to COMMIT is safe.
//
// Metrics are in the metrics subpackage: unkey_globalcounter_persist_rows_total
// counts rows by result, and unkey_globalcounter_memory_entries gauges entries,
// one per name and writer id. Snapshots dropped from a full queue appear in
// unkey_buffer_state_total.
package globalcounter
