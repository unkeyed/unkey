import { relations } from "drizzle-orm";
import { bigint, boolean, mysqlTable } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { workspaces } from "./workspaces";

/**
 * limits stores one complete set of workspace limits per workspace.
 *
 * API request throttling is nullable because NULL means no automatic workspace
 * throttle. Every other limit is explicit, including zero.
 */
export const limits = mysqlTable("limits", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull().unique(),

  /**
   * Caps monthly API usage for the workspace.
   * Unit: billable API operations per calendar month.
   */
  apiBillableOperationsCountMaxPerMonth: bigint("api_billable_operations_count_max_per_month", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps workspace API traffic when a throttle is configured.
   * Unit: requests per minute. NULL means there is no workspace-level throttle.
   */
  apiRequestsCountMaxPerMinute: bigint("api_requests_count_max_per_minute", {
    mode: "number",
    unsigned: true,
  }),

  /**
   * Caps how long request and runtime logs are retained.
   * Unit: days.
   */
  logsRetentionDaysMax: bigint("logs_retention_days_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how long audit logs are retained.
   * Unit: days.
   */
  logsAuditRetentionDaysMax: bigint("logs_audit_retention_days_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Controls whether the workspace can use team features.
   * Unit: boolean enabled/disabled.
   */
  teamEnabled: boolean("team_enabled").notNull(),

  /**
   * Caps how much CPU the workspace can allocate across all running instances.
   * Unit: whole CPU cores.
   */
  cpuCoresMax: bigint("cpu_cores_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how much CPU one runtime instance can request.
   * Unit: whole CPU cores.
   */
  cpuCoresMaxPerInstance: bigint("cpu_cores_max_per_instance", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how much memory the workspace can allocate across all running instances.
   * Unit: MiB.
   */
  memoryMibMax: bigint("memory_mib_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how much memory one runtime instance can request.
   * Unit: MiB.
   */
  memoryMibMaxPerInstance: bigint("memory_mib_max_per_instance", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how much ephemeral disk the workspace can allocate across all running
   * instances. Unit: MiB.
   */
  diskEphemeralMibMax: bigint("disk_ephemeral_mib_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how much ephemeral disk one runtime instance can request.
   * Unit: MiB.
   */
  diskEphemeralMibMaxPerInstance: bigint("disk_ephemeral_mib_max_per_instance", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how many builds the workspace can run at the same time.
   * Unit: active builds.
   */
  buildsConcurrentCountMax: bigint("builds_concurrent_count_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),

  /**
   * Caps how many custom domains the workspace can add.
   * Unit: domains.
   */
  customDomainsCountMax: bigint("custom_domains_count_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
});

export const limitsRelations = relations(limits, ({ one }) => ({
  workspace: one(workspaces, {
    relationName: "workspace_limit_relation",
    fields: [limits.workspaceId],
    references: [workspaces.id],
  }),
}));
