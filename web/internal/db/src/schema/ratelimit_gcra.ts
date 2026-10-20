import { bigint, index, mysqlTable, primaryKey } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

export const ratelimitGcra = mysqlTable(
  "ratelimit_gcra",
  {
    /**
     * Identifies one rate limit: workspace, namespace, identifier and duration,
     * length-prefixed, or their SHA-256 when that is too long.
     */
    name: caseSensitiveVarchar("name", { length: 512 }).notNull(),
    /**
     * `<random process id>_<sequence>`. A process takes a new sequence for every
     * entry it creates and never reuses one, because a rebuilt entry starting at
     * zero would be hidden by GREATEST on upsert.
     */
    writerId: caseSensitiveVarchar("writer_id", { length: 48 }).notNull(),
    /**
     * Total time this writer admitted, in nanoseconds. Only grows. Writers are
     * combined by SUM.
     */
    spentNs: bigint("spent_ns", { mode: "bigint" }).notNull(),
    /**
     * The writer's highest proposed baseline, Unix nanoseconds. Only grows.
     * Writers are combined by MAX, and reset_at = MAX(fresh_start_ns) + SUM(spent_ns).
     */
    freshStartNs: bigint("fresh_start_ns", { mode: "bigint" }).notNull(),
    /**
     * Unix nanoseconds after which this row no longer matters to its writer.
     * Cleanup deletes a name only when every row of it expired.
     */
    expiresAtNs: bigint("expires_at_ns", { mode: "bigint" }).notNull(),
    /**
     * Unix milliseconds by the database clock of the last upsert that changed the row.
     */
    updatedAtMs: bigint("updated_at_ms", { mode: "number" }).notNull(),
  },
  (table) => [
    primaryKey({ columns: [table.name, table.writerId] }),
    index("ratelimit_gcra_expires_at_ns_idx").on(table.expiresAtNs),
  ],
);
