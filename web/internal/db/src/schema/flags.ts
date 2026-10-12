import { boolean, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const flags = mysqlTable("flags", {
  pk: primaryKey(),
  /** Stable, case-sensitive identifier used by application code and workspace overrides. */
  slug: caseSensitiveVarchar("slug", { length: 128 }).notNull().unique(),
  /** Customer-facing explanation of the platform feature. */
  description: varchar("description", { length: 1024 }).notNull(),
  /** Effective value when a workspace has no override. */
  defaultValue: boolean("default_value").notNull(),
  /**
   * Whether customer-facing lists should show this flag. Impersonation views
   * should show all flags regardless. Visibility does not affect evaluation.
   */
  visible: boolean("visible").notNull().default(true),
  /**
   * Whether customers may set their workspace override to true. Customer write
   * paths must enforce this policy; internal administration is not restricted.
   */
  allowEnable: boolean("allow_enable").notNull().default(false),
  /**
   * Whether customers may set their workspace override to false. Customer write
   * paths must enforce this policy; internal administration is not restricted.
   */
  allowDisable: boolean("allow_disable").notNull().default(false),
});

export const flagOverrides = mysqlTable(
  "flag_overrides",
  {
    pk: primaryKey(),
    /** Workspace whose effective flag value this row overrides. */
    workspaceId: id("workspace_id").notNull(),
    /** References the immutable flags.slug identifier. */
    flagSlug: caseSensitiveVarchar("flag_slug", { length: 128 }).notNull(),
    /** Overrides the default even when false. Deleting this row restores the flag's default. */
    value: boolean("value").notNull(),
  },
  (table) => ({
    workspaceFlag: uniqueIndex("workspace_flag_idx").on(table.workspaceId, table.flagSlug),
  }),
);
