import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { keyAuth } from "./keyAuth";
import { deleteProtection } from "./util/delete_protection";
import { id } from "./util/id";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const apis = mysqlTable(
  "apis",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: id("id").notNull().unique(),
    name: varchar("name", { length: 256 }).notNull(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull().default(""),
    /**
     * comma separated ips
     */
    ipWhitelist: varchar("ip_whitelist", { length: 512 }),
    authType: mysqlEnum("auth_type", ["key", "jwt"]),
    keyAuthId: id("key_auth_id").unique(),

    ...lifecycleDatesMigration,
    ...deleteProtection,
  },
  (table) => [
    index("workspace_id_idx").on(table.workspaceId),
    index("apis_project_id_idx").on(table.projectId),
  ],
);

export const apisRelations = relations(apis, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [apis.workspaceId],
    references: [workspaces.id],
  }),
  keyAuth: one(keyAuth, {
    fields: [apis.keyAuthId],
    references: [keyAuth.id],
  }),
}));
