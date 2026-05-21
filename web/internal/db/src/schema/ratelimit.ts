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
 * Cross-region sharing of global rate-limit counters.
 *
 * Each region periodically flushes its own observed count for each active
 * sliding-window cell into a row keyed by (workspace, namespace, identifier,
 * duration_ms, sequence, region). Other regions read every row whose region
 * differs from their own and sum `count` across regions to derive their
 * `imported` count, which the local sliding-window math adds on top of its
 * own count to make a globally-aware deny decision.
 *
 * Sharing the actual quantity lets receivers reach the same deny decision the
 * originator did without the over-block failure modes of verdict propagation.
 *
 * The unique key includes `region` so each region writes its own row;
 * concurrent writers within a region collapse via ON DUPLICATE KEY UPDATE
 * count = GREATEST(count, VALUES(count)). Aggregation across regions is SUM
 * over the rows for the same window cell.
 */
export const ratelimitGlobalCounters = mysqlTable(
  "ratelimit_global_counters",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    // workspaceId is varchar(191) instead of the project-wide 256 because the
    // unique index spans six columns (the four key fields plus sequence and
    // region), and MySQL caps total index key size at 3072 bytes under utf8mb4.
    workspaceId: varchar("workspace_id", { length: 191 }).notNull(),
    namespace: varchar("namespace", { length: 255 }).notNull(),
    identifier: varchar("identifier", { length: 255 }).notNull(),
    durationMs: bigint("duration_ms", { mode: "number", unsigned: true }).notNull(),
    /**
     * Sliding-window sequence this row's count belongs to. Receivers apply the
     * row to the exact sequence stored here, so sliding-window decay math stays
     * consistent with the originating region.
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
     * region's sliding-window math: (sequence + 2) * duration_ms.
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
  ],
);
