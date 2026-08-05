import { relations } from "drizzle-orm";
import { bigint, boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { portalBranding } from "./portal_branding";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const portalConfigurations = mysqlTable(
  "portal_configurations",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    slug: varchar("slug", { length: 64 }).notNull(),
    appId: id("app_id"),
    keyAuthId: caseSensitiveVarchar("key_auth_id", { length: 37 }),
    enabled: boolean("enabled").notNull().default(true),
    returnUrl: varchar("return_url", { length: 500 }),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("idx_workspace_slug").on(table.workspaceId, table.slug),
    uniqueIndex("idx_app_id").on(table.appId),
    uniqueIndex("idx_key_auth_id").on(table.keyAuthId),
  ],
);

export const portalConfigurationsRelations = relations(portalConfigurations, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalConfigurations.workspaceId],
    references: [workspaces.id],
  }),
  branding: one(portalBranding, {
    fields: [portalConfigurations.id],
    references: [portalBranding.portalConfigId],
  }),
}));
