import { relations } from "drizzle-orm";
import { bigint, boolean, mysqlTable } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { workspaces } from "./workspaces";

/**
 * limits is the typed migration target for workspace limits, with one complete
 * row per workspace. During dual-write, quota remains the read source until
 * every legacy row has been backfilled and verified.
 *
 * API request throttling is nullable because NULL means no automatic workspace
 * throttle. Every other entitlement is explicit, including zero.
 */
export const limits = mysqlTable("limits", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull().unique(),
  apiBillableOperationsCountMaxPerMonth: bigint("api_billable_operations_count_max_per_month", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  apiRequestsCountMaxPerMinute: bigint("api_requests_count_max_per_minute", {
    mode: "number",
    unsigned: true,
  }),
  logsRetentionDaysMax: bigint("logs_retention_days_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  logsAuditRetentionDaysMax: bigint("logs_audit_retention_days_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  teamEnabled: boolean("team_enabled").notNull(),
  cpuMax: bigint("cpu_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  cpuMaxPerInstance: bigint("cpu_max_per_instance", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  memoryMibMax: bigint("memory_mib_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  memoryMibMaxPerInstance: bigint("memory_mib_max_per_instance", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  diskEphemeralMibMax: bigint("disk_ephemeral_mib_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  diskEphemeralMibMaxPerInstance: bigint("disk_ephemeral_mib_max_per_instance", {
    mode: "number",
    unsigned: true,
  }).notNull(),
  buildsConcurrentCountMax: bigint("builds_concurrent_count_max", {
    mode: "number",
    unsigned: true,
  }).notNull(),
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
