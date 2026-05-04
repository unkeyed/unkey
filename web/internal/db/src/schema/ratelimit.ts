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
