import { boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const flags = mysqlTable("flags", {
  pk: primaryKey(),
  slug: caseSensitiveVarchar("slug", { length: 128 }).notNull().unique(),
  description: varchar("description", { length: 1024 }).notNull(),
  defaultValue: boolean("default_value").notNull(),
  visible: boolean("visible").notNull().default(true),
  allowOptIn: boolean("allow_opt_in").notNull().default(false),
  allowOptOut: boolean("allow_opt_out").notNull().default(false),
});

export const flagOverrides = mysqlTable(
  "flag_overrides",
  {
    pk: primaryKey(),
    workspaceId: id("workspace_id").notNull(),
    flagSlug: caseSensitiveVarchar("flag_slug", { length: 128 }).notNull(),
    value: boolean("value").notNull(),
  },
  (table) => ({
    workspaceFlag: uniqueIndex("workspace_flag_idx").on(table.workspaceId, table.flagSlug),
  }),
);
