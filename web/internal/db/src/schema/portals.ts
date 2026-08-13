import { relations } from "drizzle-orm";
import { boolean, json, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { portalSessions } from "./portal_sessions";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

/**
 * Branding is 1:1 with the portal and every config read needs it, so it lives on
 * the portal row rather than in a join. JSON rather than columns because it is
 * never queried by value and is expected to grow (dark-mode logo, favicon,
 * fonts) without widening the table. Shape is validated at the API/form
 * boundary.
 */
export type PortalBranding = {
  logoUrl?: string;
  primaryColor?: string;
};

export const portals = mysqlTable(
  "portals",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    slug: varchar("slug", { length: 64 }).notNull(),
    appId: id("app_id"),
    keyAuthId: id("key_auth_id"),
    enabled: boolean("enabled").notNull().default(true),
    returnUrl: varchar("return_url", { length: 500 }),
    branding: json("branding").$type<PortalBranding>(),
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
  sessions: many(portalSessions),
}));
