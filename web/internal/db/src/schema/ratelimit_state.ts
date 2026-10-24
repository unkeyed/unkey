import { bigint, binary, index, mysqlTable, primaryKey } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";

/**
 * Global rate limit state. Each session writes its own row. The state of a
 * limit is computed from all rows of its fingerprint:
 *
 *   reset_at = MAX over rows r of
 *              (r.anchor_at_ns + SUM(consumed_ns of rows anchored at or after r))
 *
 * No row depends on another row, so deleting rows of an idle limit cannot open
 * it early.
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
     * A random id for one session of a pod on this limit. A session starts when
     * the pod creates an in-memory entry for the fingerprint, when it finds the
     * limit idle, or when the current session reached its lifetime. After that,
     * new requests go to the new session and the old row is frozen. An id is never reused. Not related to HTTP sessions or
     * portal_sessions.
     */
    sessionId: caseSensitiveVarchar("session_id", { length: 48 }).notNull(),
    /**
     * Unix nanoseconds when the session started. This session's time counts
     * from here. Set once and never changed.
     */
    anchorAtNs: bigint("anchor_at_ns", { mode: "bigint" }).notNull(),
    /**
     * Time used by the requests this session allowed, in nanoseconds. A request
     * adds cost × duration / limit.
     */
    consumedNs: bigint("consumed_ns", { mode: "bigint" }).notNull(),
    /**
     * Unix milliseconds after which the row can never change again: the anchor
     * plus the session lifetime plus a margin for writes that are still on
     * their way. Set once and never changed. After this time, cleanup may merge
     * the row with other finished rows, or delete it when the limit is idle.
     */
    compactAfterMs: bigint("compact_after_ms", { mode: "number" }).notNull(),
  },
  (table) => [
    primaryKey({
      name: "ratelimit_state_fingerprint_session_id_pk",
      columns: [table.fingerprint, table.sessionId],
    }),
    index("ratelimit_state_compact_after_ms_idx").on(table.compactAfterMs),
  ],
);
