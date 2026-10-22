import { bigint, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// cli_sessions is the credential a device login issues. The access token is a
// short-lived JWT and is not stored. The refresh token is stored only as a
// hash. previous_refresh_token_hash keeps the token just rotated away so a
// replay of that token revokes the refresh_family. expires_at is an idle
// deadline moved forward on each successful refresh. revoked_at wins over
// expiry: a revoked row must never authenticate.
export const cliSessions = mysqlTable(
  "cli_sessions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    userId: caseSensitiveVarchar("user_id", { length: 256 }).notNull(),
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
