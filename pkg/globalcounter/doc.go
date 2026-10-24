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
// its delete_after. Close stops eviction
// and rotation, drains queued snapshots, then persists the entries of the
// current and the previous writer id, including dropped snapshots, within one
// deadline.
//
// Metrics are in the metrics subpackage: unkey_globalcounter_persist_rows_total
// counts rows by result, and unkey_globalcounter_memory_entries gauges entries,
// one per name and writer id. Snapshots dropped from a full queue appear in
// unkey_buffer_state_total.
package globalcounter
