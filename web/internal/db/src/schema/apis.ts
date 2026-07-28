import { relations } from "drizzle-orm";
import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { keyAuth } from "./keyAuth";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const apis = mysqlTable(
  "apis",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),
    name: varchar("name", { length: 256 }).notNull(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 32 }).notNull(),
    projectId: caseSensitiveVarchar("project_id", { length: 32 }).notNull().default(""),
    /**
     * comma separated ips
     */
    ipWhitelist: varchar("ip_whitelist", { length: 512 }),
    authType: mysqlEnum("auth_type", ["key", "jwt"]),
    keyAuthId: caseSensitiveVarchar("key_auth_id", { length: 32 }).unique(),

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
