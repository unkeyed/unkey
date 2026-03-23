import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const portalConfigurations = mysqlTable(
  "portal_configurations",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }),
    keyAuthId: varchar("key_auth_id", { length: 64 }),
    enabled: boolean("enabled").notNull().default(true),
    returnUrl: varchar("return_url", { length: 500 }),
    ...lifecycleDates,
  },
  (table) => [
    index("idx_workspace").on(table.workspaceId),
    uniqueIndex("idx_app_id").on(table.appId),
    uniqueIndex("idx_key_auth_id").on(table.keyAuthId),
  ],
);

export const portalConfigurationsRelations = relations(portalConfigurations, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [portalConfigurations.workspaceId],
    references: [workspaces.id],
  }),
}));
