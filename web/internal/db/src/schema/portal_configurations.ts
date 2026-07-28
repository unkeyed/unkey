import { relations } from "drizzle-orm";
import { bigint, boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { portalBranding } from "./portal_branding";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const portalConfigurations = mysqlTable(
  "portal_configurations",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 32 }).notNull(),
    slug: varchar("slug", { length: 64 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 32 }),
    keyAuthId: caseSensitiveVarchar("key_auth_id", { length: 32 }),
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
