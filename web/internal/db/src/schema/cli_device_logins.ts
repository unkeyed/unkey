import { bigint, index, int, json, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const cliDeviceLogins = mysqlTable(
  "cli_device_logins",
  {
    pk: primaryKey(),
    id: id("id").notNull(),
    userCode: caseSensitiveVarchar("user_code", { length: 32 }).notNull(),
    pollIntervalSeconds: int("poll_interval_seconds").notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    status: varchar("status", { length: 32 }).notNull(),
    deviceName: varchar("device_name", { length: 512 }),
    requesterIp: varchar("requester_ip", { length: 64 }),
    requesterUserAgent: varchar("requester_user_agent", { length: 512 }),
    workspaceId: id("workspace_id"),
    approverUserId: caseSensitiveVarchar("approver_user_id", { length: 256 }),
    approverName: varchar("approver_name", { length: 256 }),
    approverRoles: json("approver_roles").$type<string[]>(),
    permissions: json("permissions").$type<string[]>(),
    keyName: varchar("key_name", { length: 256 }),
    keyId: id("key_id"),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    approvedAt: bigint("approved_at", { mode: "number" }),
  },
  (table) => ({
    idIdx: uniqueIndex("cli_device_logins_id_unique").on(table.id),
    userCodeIdx: uniqueIndex("cli_device_logins_user_code_unique").on(table.userCode),
    expiresAtIdx: index("cli_device_logins_expires_at_idx").on(table.expiresAt),
  }),
);
