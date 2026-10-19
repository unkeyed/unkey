import { bigint, index, mysqlTable, primaryKey, tinyint } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

export const counters = mysqlTable(
  "counters",
  {
    /**
     * Opaque key that identifies a counter globally. For example a rate limiter can combine
     * its namespace ID, identifier, and window identity (duration and sequence).
     * The caller must encode these components without collisions.
     */
    name: caseSensitiveVarchar("name", { length: 512 }).notNull(),
    /**
     * Identifies one writer for a short time, not a region. It has the form
     * `<random process id>_<sequence>`. A process starts a new sequence about every
     * 15 minutes, so a counter's total is the sum of its rows. Each writer owns a
     * separate row, and a process that loses its count and revision state must use a
     * new process id. The writer ID `compacted` is reserved for the row that holds
     * the merged contributions of old writers.
     */
    writerId: caseSensitiveVarchar("writer_id", { length: 48 }).notNull(),
    /**
     * This writer's absolute signed contribution, not a delta or the global
     * total. A newer snapshot can lower it to represent deductions.
     * Drizzle's number mapping is exact only within JavaScript's safe integer range.
     */
    count: bigint("count", { mode: "number" }).notNull().default(0),
    /**
     * Orders snapshots for this row independently of the count. A newer
     * snapshot must have a larger revision, even when its count decreases.
     * Publish ignores equal or older revisions; retries reuse the same snapshot.
     *
     * ```text
     * Time   Incoming (revision, count)       Stored (revision, count)
     * t0     (7, 100) ---------------------> (7, 100)
     * t1     (8, -20) ---------------------> (8, -20)  deduction
     * t2     (7, 100) ---------------------> (8, -20)  ignore late snapshot
     * t3     (8, -20) ---------------------> (8, -20)  ignore retry
     * ```
     */
    revision: bigint("revision", { mode: "bigint", unsigned: true }).notNull(),
    /**
     * Unix milliseconds after which the row may be deleted, set by the caller. It
     * is not a read TTL: reads include the row until deletion. NULL means retain
     * indefinitely; a newer snapshot replaces it, so it can extend or shorten
     * retention.
     *
     * Deletion also removes revision history. Cleanup must wait until the
     * writer can no longer publish or retry snapshots for this row.
     */
    deleteAfter: bigint("delete_after", { mode: "number" }),
    /**
     * Unix milliseconds, by the database clock, after which compaction may merge
     * the row. The persist query sets it to the time of the write plus a margin the
     * caller chooses, an hour for the global counter, and moves it only when it
     * accepts a newer revision. Once it has passed, no writer writes the row
     * anymore, and that is the only reason compaction may merge it without
     * counting it twice. Compaction moves it ahead again for a row it has nothing
     * to merge with, by the wait of the row's compaction level.
     */
    compactionAfter: bigint("compaction_after", { mode: "number" }).notNull(),
    /**
     * Back-off level of compaction. Level 0 is for a row that a writer wrote: it
     * waits for the margin of `compaction_after`, and no compaction pass ever writes
     * level 0. After compaction first handles a row, the row is at level 1 or
     * higher, and a pass looks at it again after 10 minutes times 2^level, up to a
     * year: 20 minutes at level 1, 40 minutes at level 2, and so on. A row that has
     * nothing to merge with goes up one level, so a counter that nobody writes is
     * looked at less and less often. A merged row takes the lowest level of the rows
     * it replaces, but at least 1, so a counter that still gets new rows stays at a
     * low level and is compacted often.
     */
    compactionLevel: tinyint("compaction_level", { unsigned: true }).notNull().default(0),
  },
  (table) => [
    primaryKey({ columns: [table.name, table.writerId] }),
    index("counter_delete_after").on(table.deleteAfter),
    index("counter_compaction_after").on(table.compactionAfter),
  ],
);
