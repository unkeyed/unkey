import {
  bigint,
  boolean,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  varchar,
} from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const deviceAuthorizations = mysqlTable(
  "device_authorizations",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    userCode: caseSensitiveVarchar("user_code", { length: 32 }).notNull().unique(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    status: mysqlEnum("status", ["pending", "approved", "consumed", "denied", "expired"]).notNull(),
    deviceName: varchar("device_name", { length: 512 }),
    requesterIp: varchar("requester_ip", { length: 64 }),
    requesterUserAgent: varchar("requester_user_agent", { length: 512 }),
    workspaceId: id("workspace_id"),
    approverUserId: caseSensitiveVarchar("approver_user_id", { length: 256 }),
    approverName: varchar("approver_name", { length: 256 }),
    approverRoles: json("approver_roles").$type<string[]>(),
    permissions: json("permissions").$type<string[]>(),
    sessionId: id("session_id").unique(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    approvedAt: bigint("approved_at", { mode: "number" }),
    // A non-null client_id marks an OAuth authorization request. The redirect
    // columns are null for CLI device logins.
    clientId: id("client_id"),
    redirectUri: caseSensitiveVarchar("redirect_uri", { length: 2048 }),
    redirectUriPresented: boolean("redirect_uri_presented"),
    state: caseSensitiveVarchar("state", { length: 1024 }),
    codeChallenge: caseSensitiveVarchar("code_challenge", { length: 128 }),
    codeHash: caseSensitiveVarchar("code_hash", { length: 64 }).unique(),
  },
  (table) => [index("device_authorizations_expires_at_idx").on(table.expiresAt)],
);
