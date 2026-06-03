import { bigint, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";

/**
 * deletions is the source of truth for which resources are currently in
 * their soft-delete grace window. Each row records one resource that
 * has been marked for permanent removal at delete_permanently_at.
 *
 * The row is written by the resource's MarkForDeletion VO handler (or
 * the direct dashboard tRPC for resources without a VO, like ratelimit
 * namespaces) at the same time as the resource table's `deletion_id`
 * column is set to this row's `id`. The two writes go together: the
 * `deletion_id` column is the cheap read-filter denormalization
 * (everything filters `WHERE deletion_id IS NULL`); this table is the
 * SoT for cascade correlation and the cron sweep target.
 *
 * Cascade correlation comes from delete_permanently_at: every row
 * inserted as part of one soft-delete cascade shares the exact same
 * value, computed once at the root inside a restate.Run so Restate
 * replays preserve it. A restore reads T from the root row and walks
 * descendants in this table whose delete_permanently_at matches;
 * resources deleted independently carry a different T and are not
 * reachable from the cascade walk, so they stay deleted.
 *
 * The cron sweep selects rows whose delete_permanently_at has elapsed
 * and fires the per-resource hard-delete path. After the cascade
 * completes, the row in this table is removed.
 */
export const deletions = mysqlTable(
  "deletions",
  {
    // id is the primary key — referenced by `<resource>.deletion_id`
    // on every soft-deletable table.
    id: varchar("id", { length: 64 }).notNull().primaryKey(),

    // workspace_id is denormalized from the resource so the dashboard
    // can list all scheduled deletions for a workspace in a single
    // query rather than UNION across resource tables.
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    // resource_type values are stable identifiers shared with the cron
    // dispatch and the dashboard restore dispatcher. See the Go enum
    // and the TS union for the canonical list.
    resourceType: varchar("resource_type", { length: 64 }).notNull(),
    resourceId: varchar("resource_id", { length: 256 }).notNull(),

    // delete_permanently_at doubles as the cron-sweep cutoff and as
    // the cascade correlation key. See the package doc above.
    deletePermanentlyAt: bigint("delete_permanently_at", { mode: "number" }).notNull(),
  },
  (table) => [
    // A resource can only be in one active deletion at a time.
    uniqueIndex("deletions_resource_idx").on(table.resourceType, table.resourceId),
    index("deletions_workspace_due_idx").on(table.workspaceId, table.deletePermanentlyAt),
    index("deletions_due_idx").on(table.deletePermanentlyAt),
  ],
);
