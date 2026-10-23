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
     * Gives every writer its own row, so writers never contend and a writer can
     * leave its own rows out of a read. It has the form `<random process id>_<sequence>`
     * and changes about every 15 minutes, so that a row stops changing soon after it
     * is created; that is what lets compaction merge old rows safely. A process that
     * loses its count and revision state must pick a new process id, because its
     * old revisions would reject its new snapshots. Compaction writes its merged rows
     * under ids that start with `compacted_`.
     */
    writerId: caseSensitiveVarchar("writer_id", { length: 48 }).notNull(),
    /**
     * The writer's full signed contribution, not a delta, so a lost write is
     * repaired by the next one and a repeated write cannot count twice. A newer
     * snapshot may lower it to represent deductions. Drizzle's number mapping is
     * exact only within JavaScript's safe integer range.
     */
    count: bigint("count", { mode: "number" }).notNull().default(0),
    /**
     * Orders the snapshots of this row. The count cannot order them, because a
     * deduction lowers it, so the persist query accepts only a larger revision and
     * a late or retried snapshot changes nothing.
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
     * Unix milliseconds, by the database clock, before which compaction must not
     * touch the row. Compaction may only merge a row that no writer will write
     * again; otherwise the writer's next snapshot recreates the row and its count
     * is counted twice. Writers set it to the write time plus a margin they choose
     * (an hour for the global counter), well past the time a writer id stays in
     * use. Compaction reuses the column to schedule its next look at a row it could
     * not merge, by the wait of the row's compaction level.
     */
    compactionAfter: bigint("compaction_after", { mode: "number" }).notNull(),
    /**
     * Back-off of compaction, so a counter that nobody writes costs fewer and fewer
     * passes while an active one keeps being merged. A row with nothing to merge
     * with goes up one level, and a pass waits 10 minutes times 2^level, up to a
     * year, before it looks at it alone again. A merged row takes the lowest level
     * of its sources, but at least 1, because a low level means the counter still
     * gets new rows. Level 0 is reserved for rows that a writer wrote, which wait
     * for the writer's margin instead.
     */
    compactionLevel: tinyint("compaction_level", { unsigned: true }).notNull().default(0),
  },
  (table) => [
    primaryKey({ columns: [table.name, table.writerId] }),
    index("counter_delete_after").on(table.deleteAfter),
    index("counter_compaction_after").on(table.compactionAfter),
  ],
);
