import { boolean, json, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const flags = mysqlTable("flags", {
  pk: primaryKey(),
  id: id("id").notNull().unique(),
  slug: caseSensitiveVarchar("slug", { length: 128 }).notNull().unique(),
  description: varchar("description", { length: 1024 }).notNull(),
  type: mysqlEnum("type", ["boolean", "string", "number"]).notNull(),
  defaultValue: json("default_value").$type<boolean | string | number>().notNull(),
  allowOptIn: boolean("allow_opt_in").notNull().default(false),
  allowOptOut: boolean("allow_opt_out").notNull().default(false),
});

export const workspaceFlagOverrides = mysqlTable(
  "workspace_flag_overrides",
  {
    pk: primaryKey(),
    workspaceId: id("workspace_id").notNull(),
    flagId: id("flag_id").notNull(),
    value: json("value").$type<boolean | string | number>().notNull(),
  },
  (table) => ({
    workspaceFlag: uniqueIndex("workspace_flag_idx").on(table.workspaceId, table.flagId),
  }),
);
