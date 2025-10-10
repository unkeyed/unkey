import { relations } from "drizzle-orm";
import { bigint, int, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDatesV2 } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * ClickHouse configuration for workspaces with analytics enabled.
 * Each workspace gets a dedicated user with resource quotas to prevent abuse.
 */
export const clickhouseWorkspaceSettings = mysqlTable("clickhouse_workspace_settings", {
  workspaceId: varchar("workspace_id", { length: 256 }).primaryKey(),

  // Authentication
  username: varchar("username", { length: 256 }).notNull().unique(),
  passwordEncrypted: text("password_encrypted").notNull(),

  // Quota window configuration
  quotaDurationSeconds: int("quota_duration_seconds").notNull().default(3_600), // 1 hour
  maxQueriesPerWindow: int("max_queries_per_window").notNull().default(1_000),
  maxExecutionTimePerWindow: int("max_execution_time_per_window").notNull().default(1_800), // 30 min total

  // Per-query limits (prevent cluster crashes)
  maxQueryExecutionTime: int("max_query_execution_time").notNull().default(30), // seconds
  maxQueryMemoryBytes: bigint("max_query_memory_bytes", { mode: "number" })
    .notNull()
    .default(1_000_000_000), // 1GB
  maxQueryResultRows: int("max_query_result_rows").notNull().default(10_000),
  maxRowsToRead: bigint("max_rows_to_read", { mode: "number" }).notNull().default(10_000_000), // 10M rows

  ...lifecycleDatesV2,
});

export const clickhouseWorkspaceSettingsRelations = relations(
  clickhouseWorkspaceSettings,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [clickhouseWorkspaceSettings.workspaceId],
      references: [workspaces.id],
    }),
  }),
);
