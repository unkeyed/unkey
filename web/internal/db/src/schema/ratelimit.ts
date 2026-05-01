import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, unique, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const ratelimitNamespaces = mysqlTable(
  "ratelimit_namespaces",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 512 }).notNull(),

    ...lifecycleDatesMigration,
  },
  (table) => {
    return {
      uniqueNamePerWorkspaceIdx: unique("unique_name_per_workspace_idx").on(
        table.workspaceId,
        table.name,
      ),
    };
  },
);

export const ratelimitNamespaceRelations = relations(ratelimitNamespaces, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [ratelimitNamespaces.workspaceId],
    references: [workspaces.id],
  }),
  overrides: many(ratelimitOverrides),
}));

export const ratelimitOverrides = mysqlTable(
  "ratelimit_overrides",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
    identifier: varchar("identifier", { length: 512 }).notNull(),

    limit: bigint("limit", { mode: "number", unsigned: true }).notNull(),
    /**
     * window duration in milliseconds
     */
    duration: bigint("duration", { mode: "number", unsigned: true }).notNull(),
    ...lifecycleDatesMigration,
  },
  (table) => {
    return {
      uniqueIdentifierPerNamespace: unique("unique_identifier_per_namespace_idx").on(
        table.namespaceId,
        table.identifier,
      ),
    };
  },
);
export const ratelimitOverridesRelations = relations(ratelimitOverrides, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [ratelimitOverrides.workspaceId],
    references: [workspaces.id],
  }),
  namespace: one(ratelimitNamespaces, {
    fields: [ratelimitOverrides.namespaceId],
    references: [ratelimitNamespaces.id],
  }),
}));

/**
 * Cross-region propagation of rate-limit denials.
 *
 * When a node denies a request for the first time in a window, it writes a row
 * here. Every node in every region polls this table periodically and uses each
 * row to inflate its local counter for the matching key, so the denial bleeds
 * into the receiving region's sliding-window math without a Redis roundtrip.
 *
 * Rows are short-lived: cleanup is done by an external cron deleting where
 * `expires_at < now`.
 *
 * The unique key (workspaceId, namespace, identifier, durationMs) lets the
 * write path use ON DUPLICATE KEY UPDATE to dedup concurrent writers from
 * multiple regions seeing the same offender at the same time.
 */
export const ratelimitBlocklist = mysqlTable(
  "ratelimit_blocklist",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    // workspaceId is varchar(191) instead of the project-wide 256 because the
    // unique index spans five columns (the four key fields plus sequence) and
    // MySQL caps total index key size at 3072 bytes under utf8mb4. Real
    // workspace IDs are ~22 chars, so 191 is comfortable.
    workspaceId: varchar("workspace_id", { length: 191 }).notNull(),
    namespace: varchar("namespace", { length: 255 }).notNull(),
    identifier: varchar("identifier", { length: 255 }).notNull(),
    durationMs: bigint("duration_ms", { mode: "number", unsigned: true }).notNull(),
    /**
     * Sliding-window sequence the originating denial fell into. Part of the
     * unique key so each (key, sequence) gets its own row — sequence
     * advancement creates a new row rather than mutating an existing one.
     * Receivers inflate the exact sequence stored here, not whichever
     * sequence their clock currently maps to, so the sliding-window decay
     * math matches the originating region's state.
     */
    sequence: bigint("sequence", { mode: "number" }).notNull(),
    limit: bigint("limit", { mode: "number", unsigned: true }).notNull(),
    /**
     * Unix-millis cutoff after which the row is no longer relevant to any
     * region's sliding-window math. Computed application-side as
     * (sequence + 2) * duration_ms — end of the window after the originating
     * one, since the inflated counter contributes as cur in `sequence` and
     * as prev in `sequence + 1`. Used by the cleanup cron only.
     */
    expiresAt: bigint("expires_at", { mode: "number", unsigned: true }).notNull(),
  },
  (table) => [
    uniqueIndex("unique_propagation_key").on(
      table.workspaceId,
      table.namespace,
      table.identifier,
      table.durationMs,
      table.sequence,
    ),
    index("expires_at_idx").on(table.expiresAt),
  ],
);

/**
 * Cross-region sharing of sliding-window counts (G-Counter).
 *
 * Each region periodically flushes its own observed count for each active
 * sliding-window cell into a row keyed by (workspace, namespace, identifier,
 * duration_ms, sequence, region). Other regions read every row whose region
 * differs from their own and sum `count` across regions to derive their
 * `imported` count, which the local sliding-window math adds on top of its
 * own count to make a globally-aware deny decision.
 *
 * The shape replaces ratelimit_blocklist's verdict-shaped propagation. Where
 * blocklist said "this region denied, you should too," window_counts says
 * "this region has seen N requests, factor that into your math." Sharing the
 * actual quantity lets receivers reach the same deny decision the originator
 * did without the over-block failure modes of inflating to limit on denial.
 *
 * The unique key includes `region` so each region writes its own row;
 * concurrent writers within a region collapse via ON DUPLICATE KEY UPDATE
 * count = GREATEST(count, VALUES(count)). Aggregation across regions is SUM
 * over the rows for the same window cell.
 */
export const ratelimitWindowCounts = mysqlTable(
  "ratelimit_window_counts",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    // workspaceId at varchar(191) for the same reason as ratelimitBlocklist:
    // the unique index now spans six columns (the four key fields plus
    // sequence and region), and MySQL caps total index key size at 3072
    // bytes under utf8mb4.
    workspaceId: varchar("workspace_id", { length: 191 }).notNull(),
    namespace: varchar("namespace", { length: 255 }).notNull(),
    identifier: varchar("identifier", { length: 255 }).notNull(),
    durationMs: bigint("duration_ms", { mode: "number", unsigned: true }).notNull(),
    /**
     * Sliding-window sequence this row's count belongs to. Same role as in
     * ratelimit_blocklist: receivers apply the row to the exact sequence
     * stored here, so sliding-window decay math stays consistent with the
     * originating region.
     */
    sequence: bigint("sequence", { mode: "number" }).notNull(),
    /**
     * Region identifier of the writing fleet. Sourced from UNKEY_REGION at
     * process start. Part of the unique key so each region's count lives in
     * its own row; aggregation is SUM across rows for the same window cell.
     *
     * Capped at varchar(48) so the unique index (which spans the four key
     * fields plus sequence and region) stays under MySQL's 3072-byte limit
     * for utf8mb4. Real region tags (aws region IDs, datacenter codes) sit
     * comfortably under this; the cap exists only to bound the index size.
     */
    region: varchar("region", { length: 48 }).notNull(),
    /**
     * Region's observed count for this window cell. Monotonic per region
     * within a sequence (own count only ever grows). Concurrent writers
     * within the region collapse via ON DUPLICATE KEY UPDATE
     * count = GREATEST(count, VALUES(count)).
     */
    count: bigint("count", { mode: "number", unsigned: true }).notNull(),
    /**
     * Unix-millis cutoff after which the row is no longer relevant to any
     * region's sliding-window math. Same derivation as
     * ratelimit_blocklist.expires_at: (sequence + 2) * duration_ms.
     */
    expiresAt: bigint("expires_at", { mode: "number", unsigned: true }).notNull(),
    /**
     * Unix-millis of the most recent flush that touched this row. Used by
     * receivers to skip rows older than the previous sync's cutoff so each
     * sync only processes incremental updates. The application sets this on
     * every upsert; MySQL never updates it implicitly.
     */
    updatedAt: bigint("updated_at", { mode: "number", unsigned: true }).notNull(),
  },
  (table) => [
    uniqueIndex("unique_window_region").on(
      table.workspaceId,
      table.namespace,
      table.identifier,
      table.durationMs,
      table.sequence,
      table.region,
    ),
    index("expires_at_idx").on(table.expiresAt),
    index("lookup_idx").on(
      table.workspaceId,
      table.namespace,
      table.identifier,
      table.durationMs,
      table.sequence,
    ),
  ],
);
