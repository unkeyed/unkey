import {
  bigint,
  boolean,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDatesV2, softDelete } from "./util/lifecycle_dates";

/**
 * Marketplace-installed extension instances.
 *
 * One row per (project, extension_slug, instance_name). Multiple instances of
 * the same extension are supported (e.g. two log drains pointing at different
 * Axiom datasets) so `instance_name` is part of the uniqueness key.
 *
 * `config` holds the raw form state from the install wizard so the dashboard
 * can re-render the configure tab without per-extension columns. Provider
 * services (e.g. log_drains) project the parts they care about into their
 * own typed tables and join back to this row by id.
 */
export const installationStatus = mysqlEnum("status", [
  "active",
  "degraded",
  "disabled",
  "verifying",
  "failed",
]);

export const extensionInstallations = mysqlTable(
  "extension_installations",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),
    extensionSlug: varchar("extension_slug", { length: 128 }).notNull(),
    instanceName: varchar("instance_name", { length: 256 }).notNull(),
    /** Raw config form state. Shape matches lib/extensions/registry#ExtensionConfigState. */
    config: json("config").$type<Record<string, string | boolean | string[]>>().notNull(),
    status: installationStatus.notNull().default("active"),
    oauthConnected: boolean("oauth_connected").notNull().default(false),
    /** Last time the extension produced a noticeable event (e.g. forwarded a log batch). */
    lastEventAt: bigint("last_event_at", { mode: "number" }),
    ...lifecycleDatesV2,
    ...softDelete,
  },
  (table) => [
    index("workspace_idx").on(table.workspaceId),
    index("project_idx").on(table.projectId),
    index("extension_slug_idx").on(table.extensionSlug),
    // One instance name per (project, extension). Soft-deleted rows keep the
    // slot occupied; reuse requires hard cleanup or a fresh name.
    uniqueIndex("unique_project_extension_instance_idx").on(
      table.projectId,
      table.extensionSlug,
      table.instanceName,
    ),
  ],
);
