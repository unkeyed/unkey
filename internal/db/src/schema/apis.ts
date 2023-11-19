import { relations } from "drizzle-orm";
import { mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { auditLogs } from "./audit";
import { keyAuth } from "./keyAuth";
import { workspaces } from "./workspaces";

export const apis = mysqlTable(
  "apis",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    name: varchar("name", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // comma separated ips or cidr blocks
    ipWhitelist: varchar("ip_whitelist", { length: 512 }),

    authType: mysqlEnum("auth_type", ["key", "jwt"]),
    keyAuthId: varchar("key_auth_id", { length: 256 }),
  },
  (table) => ({
    keyAuthIdIndex: uniqueIndex("key_auth_id_idx").on(table.keyAuthId),
  }),
);

export const apisRelations = relations(apis, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [apis.workspaceId],
    references: [workspaces.id],
  }),
  keyAuth: one(keyAuth, {
    fields: [apis.keyAuthId],
    references: [keyAuth.id],
  }),
  auditLogs: many(auditLogs),
}));
