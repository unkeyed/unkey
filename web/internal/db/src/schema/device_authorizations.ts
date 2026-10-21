import { bigint, index, json, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// device_authorizations is the pending login a device starts and a dashboard
// user approves. It is not CLI-specific: any device flow that shows a user
// code and waits for approval can use the same row. No secrets are stored.
// The poll id is the unguessable handle; user_code is the short code the
// person compares on the approval page.
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
  },
  (table) => [index("device_authorizations_expires_at_idx").on(table.expiresAt)],
);
