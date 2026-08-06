import { relations } from "drizzle-orm";
import { boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const portals = mysqlTable(
  "portals",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    slug: varchar("slug", { length: 64 }).notNull(),
    appId: id("app_id"),
    keyspaceId: id("keyspace_id"),
    enabled: boolean("enabled").notNull().default(true),
    returnUrl: varchar("return_url", { length: 500 }),
    logoUrl: varchar("logo_url", { length: 500 }),
    primaryColor: varchar("primary_color", { length: 7 }),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("idx_workspace_slug").on(table.workspaceId, table.slug),
    uniqueIndex("idx_app_id").on(table.appId),
    uniqueIndex("idx_keyspace_id").on(table.keyspaceId),
  ],
);

export const portalsRelations = relations(portals, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portals.workspaceId],
    references: [workspaces.id],
  }),
}));
