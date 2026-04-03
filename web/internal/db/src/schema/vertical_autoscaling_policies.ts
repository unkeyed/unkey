import { relations } from "drizzle-orm";
import { bigint, index, int, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * A reusable vertical autoscaling policy.
 *
 * Other tables (e.g. app_regional_settings) reference this via verticalAutoscalingPolicyId.
 * If no policy is referenced, static resource requests are used (limits / 4).
 *
 * A workload uses either a horizontal or vertical autoscaling policy, never both.
 */
export const verticalAutoscalingPolicies = mysqlTable(
  "vertical_autoscaling_policies",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    updateMode: mysqlEnum("update_mode", ["off", "initial", "recreate", "in_place_or_recreate"])
      .notNull()
      .default("off"),

    controlledResources: mysqlEnum("controlled_resources", ["cpu", "memory", "both"])
      .notNull()
      .default("both"),

    controlledValues: mysqlEnum("controlled_values", ["requests", "requests_and_limits"])
      .notNull()
      .default("requests"),

    // Resource bounds for VPA recommendations. null = no bound.
    cpuMinMillicores: int("cpu_min_millicores", { unsigned: true }),
    cpuMaxMillicores: int("cpu_max_millicores", { unsigned: true }),
    memoryMinMib: int("memory_min_mib", { unsigned: true }),
    memoryMaxMib: int("memory_max_mib", { unsigned: true }),

    ...lifecycleDates,
  },
  (table) => [index("workspace_idx").on(table.workspaceId)],
);

export const verticalAutoscalingPoliciesRelations = relations(
  verticalAutoscalingPolicies,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [verticalAutoscalingPolicies.workspaceId],
      references: [workspaces.id],
    }),
  }),
);
