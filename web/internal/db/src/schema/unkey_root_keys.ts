import { bigint, boolean, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const unkeyRootKeys = mysqlTable(
  "unkey_root_keys",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    hash: caseSensitiveVarchar("hash", { length: 256 }).notNull().unique(),
    name: varchar("name", { length: 256 }),
    prefix: varchar("prefix", { length: 16 }).notNull(),
    start: varchar("start", { length: 256 }).notNull(),
    end: varchar("end", { length: 4 }).notNull(),
    enabled: boolean("enabled").notNull(),
    expires: bigint("expires", { mode: "number" }),
    // IPv4 or IPv6 addresses and CIDR ranges, such as "192.0.2.1",
    // "2001:db8::1", "192.0.2.0/24", or "2001:db8::/32".
    // Null or an empty array allows any source IP address.
    ipAllowlist: json("ip_allowlist").$type<string[]>(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
    lastUsedAt: bigint("last_used_at", { mode: "number", unsigned: true }).notNull().default(0),
    deletedAt: bigint("deleted_at", { mode: "number" }),
  },
  (table) => ({
    workspaceIdIdx: index("workspace_id_idx").on(table.workspaceId),
  }),
);
