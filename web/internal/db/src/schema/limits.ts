import { relations } from "drizzle-orm";
import { bigint, boolean, int, mysqlTable, smallint } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { workspaces } from "./workspaces";

/**
 * limits stores one complete set of workspace limits per workspace.
 *
 * API request throttling is nullable because NULL means no automatic workspace
 * throttle. Every other limit is explicit, including zero.
 *
 * Column names should read from most significant to least significant:
 * `<resource>_<measurement>_<unit>_max[_scope]`.
 *
 * Units describe what the number measures. Qualifiers describe where or when
 * the limit applies. Put max after the unit being capped, and put scope
 * qualifiers after max only when they are needed. Boolean feature gates should
 * use `<feature>_enabled`.
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
  apiRequestsCountMaxPerMinute: int("api_requests_count_max_per_minute", { unsigned: true }),

  /**
   * Caps how long request and runtime logs are retained.
   * Unit: days.
   */
  logsRetentionDaysMax: smallint("logs_retention_days_max", { unsigned: true }).notNull(),

  /**
   * Caps how long audit logs are retained.
   * Unit: days.
   */
  logsAuditRetentionDaysMax: smallint("logs_audit_retention_days_max", {
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
  cpuCoresMax: int("cpu_cores_max", { unsigned: true }).notNull(),

  /**
   * Caps how much CPU one runtime instance can request.
   * Unit: whole CPU cores.
   */
  cpuCoresMaxPerInstance: int("cpu_cores_max_per_instance", { unsigned: true }).notNull(),

  /**
   * Caps how much memory the workspace can allocate across all running instances.
   * Unit: MiB.
   */
  memoryMibMax: int("memory_mib_max", { unsigned: true }).notNull(),

  /**
   * Caps how much memory one runtime instance can request.
   * Unit: MiB.
   */
  memoryMibMaxPerInstance: int("memory_mib_max_per_instance", { unsigned: true }).notNull(),

  /**
   * Caps how much ephemeral disk the workspace can allocate across all running
   * instances. Unit: MiB.
   */
  diskEphemeralMibMax: int("disk_ephemeral_mib_max", { unsigned: true }).notNull(),

  /**
   * Caps how much ephemeral disk one runtime instance can request.
   * Unit: MiB.
   */
  diskEphemeralMibMaxPerInstance: int("disk_ephemeral_mib_max_per_instance", {
    unsigned: true,
  }).notNull(),

  /**
   * Caps how many builds the workspace can run at the same time.
   * Unit: active builds.
   */
  buildsConcurrentCountMax: smallint("builds_concurrent_count_max", { unsigned: true }).notNull(),

  // Add these when build workers read limits from this table.
  //
  // Caps how long one build can run.
  // Unit: minutes.
  // buildsDurationMinutesMax: smallint("builds_duration_minutes_max", { unsigned: true }).notNull(),
  //
  // Caps how much CPU one build machine can use.
  // Unit: whole CPU cores.
  // buildsMachineCpuCoresMax: smallint("builds_machine_cpu_cores_max", {
  //   unsigned: true,
  // }).notNull(),
  //
  // Caps how much memory one build machine can use.
  // Unit: MiB.
  // buildsMachineMemoryMibMax: int("builds_machine_memory_mib_max", { unsigned: true }).notNull(),
  //
  // Caps how much build cache the workspace can keep.
  // Unit: GiB.
  // buildsCacheGibMax: smallint("builds_cache_gib_max", { unsigned: true }).notNull(),
  //
  // Caps how long build cache entries are retained.
  // Unit: days.
  // buildsCacheRetentionDaysMax: smallint("builds_cache_retention_days_max", {
  //   unsigned: true,
  // }).notNull(),

  /**
   * Caps how many custom domains the workspace can add.
   * Unit: domains.
   */
  customDomainsCountMax: int("custom_domains_count_max", { unsigned: true }).notNull(),

  /**
   * Caps how many replicas autoscaling can run for one app.
   * Unit: replicas.
   */
  autoscalingReplicasMax: smallint("autoscaling_replicas_max", { unsigned: true }).notNull(),
});

export const limitsRelations = relations(limits, ({ one }) => ({
  workspace: one(workspaces, {
    relationName: "workspace_limit_relation",
    fields: [limits.workspaceId],
    references: [workspaces.id],
  }),
}));
