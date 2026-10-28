import { bigint, binary, index, mysqlTable, primaryKey } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

/**
 * Global rate limit state. Each limit splits time into buckets on a grid that
 * all pods share, and each pod writes one row per bucket it used. The state of
 * a limit is computed from all rows of its fingerprint:
 *
 *   bucket_reset_at = MAX(bucket_start_ns + SUM(consumed_ns), MAX(reset_at_ns))
 *                     over the rows of one bucket
 *   reset_at        = MAX over buckets b of
 *                     (bucket_reset_at of b + SUM(consumed_ns of newer buckets))
 *
 * MySQL refuses writes to a bucket once its compact_after_ns has passed. Then
 * cleanup merges all closed buckets into one base row, or deletes them when the
 * limit is idle.
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
     * Unix nanoseconds when the bucket starts. Buckets are max(1s,
     * duration / 1000) wide. 0 marks the base row that holds
     * all merged buckets.
     */
    bucketStartNs: bigint("bucket_start_ns", { mode: "bigint" }).notNull(),
    /**
     * A random id of the API process that wrote the row. A new process gets a
     * new id. Empty for the base row.
     */
    podId: caseSensitiveVarchar("pod_id", { length: 48 }).notNull(),
    /**
     * Time used by the requests this pod allowed in this bucket, in
     * nanoseconds. A request adds cost × duration / limit.
     */
    consumedNs: bigint("consumed_ns", { mode: "bigint" }).notNull(),
    /**
     * Unix nanoseconds when this row's requests would be free if they were the
     * only ones of the limit.
     */
    resetAtNs: bigint("reset_at_ns", { mode: "bigint" }).notNull(),
    /**
     * Unix nanoseconds after which MySQL refuses writes to the row and cleanup
     * may merge or delete it.
     */
    compactAfterNs: bigint("compact_after_ns", { mode: "bigint" }).notNull(),
  },
  (table) => [
    primaryKey({
      name: "ratelimit_state_fingerprint_bucket_start_ns_pod_id_pk",
      columns: [table.fingerprint, table.bucketStartNs, table.podId],
    }),
    index("ratelimit_state_compact_after_ns_idx").on(table.compactAfterNs),
  ],
);
