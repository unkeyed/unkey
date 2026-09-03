import { relations } from "drizzle-orm";
import { boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { portalSessions } from "./portal_sessions";
import { projects } from "./projects";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

/**
 * Branding is 1:1 with the portal and every config read needs it, so it lives on
 * the portal row rather than in a join. Stored as discrete columns rather than
 * JSON so the database enforces the type and length of each field, and so a
 * malformed value cannot reach the UI as a bad CSS variable. Adding a branding
 * field is a migration; that cost is accepted in exchange for the guarantees.
 */
export const portals = mysqlTable(
  "portals",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    // The empty default keeps existing rows valid during rollout. New writes
    // always set the owning project.
    projectId: id("project_id").notNull().default(""),
    slug: varchar("slug", { length: 64 }).notNull(),
    // What end users see in the portal header and page titles. Held on the row
    // rather than derived from the mapped app or keyspace, which has no name of
    // its own that the customer controls.
    displayName: varchar("display_name", { length: 64 }).notNull(),
    appId: id("app_id"),
    keyAuthId: id("key_auth_id"),
    enabled: boolean("enabled").notNull().default(true),
    logoUrl: varchar("logo_url", { length: 500 }),
    // Hex colour including the leading '#', e.g. "#3b82f6".
    primaryColor: varchar("primary_color", { length: 7 }),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("idx_workspace_slug").on(table.workspaceId, table.slug),
    uniqueIndex("idx_app_id").on(table.appId),
    uniqueIndex("idx_key_auth_id").on(table.keyAuthId),
  ],
);

export const portalsRelations = relations(portals, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [portals.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [portals.projectId],
    references: [projects.id],
  }),
  sessions: many(portalSessions),
}));
