import { relations } from "drizzle-orm";
import { bigint, boolean, int, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { workspaces } from "./workspaces";

/**
 * quotas represents the resource allocation and retention limits for workspaces.
 *
 * Each workspace has a single quota record that defines its operational boundaries,
 * including the maximum number of requests allowed per month and data retention policies.
 * These settings control service availability and billing for the workspace.
 */
export const quotas = mysqlTable("quota", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

  /**
   * workspaceId is the primary identifier for the quota record,
   * matching the ID of the workspace it belongs to.
   */
  workspaceId: varchar("workspace_id", { length: 256 }).notNull().unique(),

  /**
   * requestsPerMonth specifies the maximum number of billable API requests
   * the workspace can make in a calendar month.
   * Exceeding this limit may result in throttling or nudges to upgrade their plan.
   * Default value is 0, indicating no requests are allowed without explicit configuration.
   */
  requestsPerMonth: bigint("requests_per_month", { mode: "number" }).notNull().default(0),

  /**
   * logsRetentionDays defines how many days operational logs will be stored
   * before automatic deletion.
   */
  logsRetentionDays: int("logs_retention_days").notNull().default(0),

  /**
   * auditLogsRetentionDays defines how many days audit logs will be stored
   * before automatic deletion. Audit logs contain security-relevant events
   * and may have different retention requirements than operational logs.
   */
  auditLogsRetentionDays: int("audit_logs_retention_days").notNull().default(0),

  /**
   * team indicates whether the workspace has team collaboration features enabled.
   * When true, the workspace supports multiple users with different roles and permissions.
   * When false, the workspace operates as a personal workspace with single-user access.
   * Default value is false, requiring explicit upgrade to enable team features.
   */
  team: boolean("team").notNull().default(false),
});
export const quotasRelations = relations(quotas, ({ one }) => ({
  workspace: one(workspaces, {
    relationName: "workspace_quota_relation",
    fields: [quotas.workspaceId],
    references: [workspaces.id],
  }),
}));
