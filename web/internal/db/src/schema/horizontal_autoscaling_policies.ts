import { relations } from "drizzle-orm";
import { bigint, index, int, mysqlTable, tinyint, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * A reusable horizontal autoscaling policy.
 *
 * Other tables (e.g. app_regional_settings, sentinels) reference this via horizontalAutoscalingPolicyId.
 * If no policy is referenced, static replica counts are used.
 */
export const horizontalAutoscalingPolicies = mysqlTable(
  "horizontal_autoscaling_policies",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

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
