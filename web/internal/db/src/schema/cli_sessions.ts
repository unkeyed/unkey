import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const cliSessions = mysqlTable(
  "cli_sessions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    userId: caseSensitiveVarchar("user_id", { length: 256 }).notNull(),
    userName: varchar("user_name", { length: 256 }),
    permissions: json("permissions").$type<string[]>().notNull(),
    deviceName: varchar("device_name", { length: 512 }),
    refreshTokenHash: caseSensitiveVarchar("refresh_token_hash", { length: 256 })
      .notNull()
      .unique(),
    previousRefreshTokenHash: caseSensitiveVarchar("previous_refresh_token_hash", {
      length: 256,
    }).unique(),
    refreshFamily: caseSensitiveVarchar("refresh_family", { length: 64 }).notNull(),
    rotatedAt: bigint("rotated_at", { mode: "number" }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    lastUsedAt: bigint("last_used_at", { mode: "number" }).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }).notNull(),
    revokedAt: bigint("revoked_at", { mode: "number" }),
    // Null for sessions held by the Unkey CLI.
    clientId: id("client_id"),
  },
  (table) => [
    index("cli_sessions_workspace_user_created_idx").on(
      table.workspaceId,
      table.userId,
      table.createdAt,
    ),
    index("cli_sessions_refresh_family_idx").on(table.refreshFamily),
    index("cli_sessions_expires_at_idx").on(table.expiresAt),
  ],
);
