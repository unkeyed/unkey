import { bigint, boolean, mysqlTable, primaryKey, varchar } from "drizzle-orm/mysql-core";

// One cursor per (drain, group). A drain belongs to N groups — one per
// (workspace, project, environment, source) tuple — and each source has
// its own (inserted_at, last_id) timeline, so a drain_id-only PK would
// let one source's tail overshoot the other's and stall the slower
// source's processGroup at "fetch returns 0 rows" forever. The PK is
// the (drain_id, group_key) tuple so each source's cursor advances
// independently.
//
// `blocked` is flipped on once consecutive failures exceed
// PauseAfterFailures. The MIN-cursor query filters blocked rows out, so
// the group's read watermark advances past the failing drain
// immediately. Resume is a dashboard action that clears the flag.
export const logDrainCursors = mysqlTable(
  "log_drain_cursors",
  {
    drainId: varchar("drain_id", { length: 64 }).notNull(),
    groupKey: varchar("group_key", { length: 128 }).notNull(),

    // (time_ms, last_id) is the cursor watermark. time_ms is `inserted_at`
    // for runtime logs and `time` for request logs. last_id is the
    // source's stable per-row id — `log_id` for runtime (Vector-minted
    // "log_<16 hex chars>" UUID-v7-shaped identifier) and `request_id`
    // for sentinel requests. Stored as a string so the cursor predicate
    // compares stored ClickHouse columns directly instead of computing
    // a fingerprint hash that would block sort-key prune.
    timeMs: bigint("time_ms", { mode: "number" }).notNull(),
    // Sized to comfortably hold any per-row id any source mints —
    // sentinel `request_id` is `req_<drain_id>_<seq>_<unix_nanos>`
    // (~46 chars today), runtime `log_id` is `log_<16 hex>` (20 chars
    // today). 256 leaves room for future provider-issued id schemes
    // (longer hashes, namespaced ids) without another migration.
    lastId: varchar("last_id", { length: 256 }).notNull().default(""),

    blocked: boolean("blocked").notNull().default(false),
    blockedReason: varchar("blocked_reason", { length: 256 }),

    updatedAt: bigint("updated_at", { mode: "number" })
      .notNull()
      .$onUpdateFn(() => Date.now()),
  },
  (table) => ({
    pk: primaryKey({ columns: [table.drainId, table.groupKey] }),
  }),
);
