import { relations } from "drizzle-orm";
import { bigint, index, int, mysqlTable, tinyint } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * A reusable horizontal autoscaling policy.
 *
 * Other tables (e.g. app_regional_settings) reference this via horizontalAutoscalingPolicyId.
 * If no policy is referenced, static replica counts are used.
 */
export const horizontalAutoscalingPolicies = mysqlTable(
  "horizontal_autoscaling_policies",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),

    workspaceId: caseSensitiveVarchar("workspace_id", { length: 32 }).notNull(),

    replicasMin: int("replicas_min").notNull(),
    replicasMax: int("replicas_max").notNull(),

    // 0-100, percentage thresholds that trigger scaling. null = not used as a signal.
    memoryThreshold: tinyint("memory_threshold"),
    cpuThreshold: tinyint("cpu_threshold"),
    rpsThreshold: tinyint("rps_threshold"),

    ...lifecycleDates,
  },
  (table) => [index("workspace_idx").on(table.workspaceId)],
);

export const horizontalAutoscalingPoliciesRelations = relations(
  horizontalAutoscalingPolicies,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [horizontalAutoscalingPolicies.workspaceId],
      references: [workspaces.id],
    }),
  }),
);
