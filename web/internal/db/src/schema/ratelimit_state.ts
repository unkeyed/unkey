import { bigint, binary, double, index, mysqlTable, primaryKey } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

/**
 * Global rate limit state. The level of a limit decays exponentially with
 * the limit's duration as time constant. Time is split into periods of
 * 10 × duration on a grid that all pods share, and each pod keeps one row per
 * period it used. The level of a limit at `now` is
 *
 *   level = SUM(scaled_sum × e^((period_start_ns − now) / duration))
 *           over all rows of its fingerprint
 *
 * scaled_sum only grows, so writes keep the greater value, and the level is
 * the same however the writes of the pods are ordered. Cleanup deletes rows
 * once their delete_after_ns has passed.
 */
export const ratelimitState = mysqlTable(
  "ratelimit_state",
  {
    /**
     * SHA-256 of the limit's scopes: workspace, namespace, identifier and
     * duration. A hash keeps the key small and fixed in size, however long the
     * identifier is. The limit value is not part of it, so changing a limit
     * keeps what was consumed.
     */
    fingerprint: binary("fingerprint", { length: 32 }).notNull(),
    /**
     * Unix nanoseconds when the period starts, a multiple of 10 × duration.
     */
    periodStartNs: bigint("period_start_ns", { mode: "bigint" }).notNull(),
    /**
     * A random id of the API process that wrote the row. A new process gets a
     * new id.
     */
    podId: caseSensitiveVarchar("pod_id", { length: 48 }).notNull(),
    /**
     * SUM(cost × e^((t − period_start_ns) / duration)) over the requests at
     * time t that this pod allowed in this period. At most cost × e^10 per
     * request, so a double keeps full precision.
     */
    scaledSum: double("scaled_sum").notNull(),
    /**
     * Unix nanoseconds after which the row's level is below 0.01 units for
     * every pod, so cleanup may delete it.
     */
    deleteAfterNs: bigint("delete_after_ns", { mode: "bigint" }).notNull(),
  },
  (table) => [
    primaryKey({
      name: "ratelimit_state_fingerprint_period_start_ns_pod_id_pk",
      columns: [table.fingerprint, table.periodStartNs, table.podId],
    }),
    index("ratelimit_state_delete_after_ns_idx").on(table.deleteAfterNs),
  ],
);
