import { relations } from "drizzle-orm";
import { index, int, mysqlTable, tinyint } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
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
    pk: primaryKey(),
    id: id("id").notNull().unique(),

    workspaceId: id("workspace_id").notNull(),

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
