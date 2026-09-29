import { bigint, boolean, datetime, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const unkeyRootKeys = mysqlTable(
  "unkey_root_keys",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    keyAuthId: id("key_auth_id").notNull(),
    forWorkspaceId: id("for_workspace_id").notNull(),
    hash: caseSensitiveVarchar("hash", { length: 256 }).notNull().unique(),
    name: varchar("name", { length: 256 }),
    prefix: varchar("prefix", { length: 16 }).notNull(),
    start: varchar("start", { length: 256 }).notNull(),
    end: varchar("end", { length: 4 }).notNull(),
    enabled: boolean("enabled").notNull(),
    expires: datetime("expires", { fsp: 3 }),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    deletedAt: bigint("deleted_at", { mode: "number" }),
  },
  (table) => ({
    forWorkspaceIdIdx: index("for_workspace_id_idx").on(table.forWorkspaceId),
  }),
);
