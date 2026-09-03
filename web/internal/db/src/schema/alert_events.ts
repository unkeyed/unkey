import { relations } from "drizzle-orm";
import { bigint, double, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { deployments } from "./deployments";
import { environments } from "./environments";
import { projects } from "./projects";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const alertMetrics = [
  "error_5xx",
  "error_4xx",
  "requests",
  "requests_drop",
  "egress_bytes",
  "cpu_seconds",
  "memory_utilization",
  "oom_killed",
  "crash_loop",
] as const;

export type AlertMetric = (typeof alertMetrics)[number];

export const alertStatuses = ["open", "resolved"] as const;

export type AlertStatus = (typeof alertStatuses)[number];

// Alerts belong to an app and environment rather than a deployment because a
// production anomaly can continue across deployments. The deployment that was
// live when the alert fired remains attached as diagnostic context.
export const alertEvents = mysqlTable(
  "alert_events",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),
    deploymentId: id("deployment_id"),
    metric: mysqlEnum("metric", alertMetrics).notNull(),
    status: mysqlEnum("status", alertStatuses).notNull().default("open"),
    firedAt: bigint("fired_at", { mode: "number" }).notNull(),
    lastSeenAt: bigint("last_seen_at", { mode: "number" }).notNull(),
    resolvedAt: bigint("resolved_at", { mode: "number" }),
    resolutionMessage: varchar("resolution_message", { length: 1000 }),
    observedValue: double("observed_value").notNull(),
    // The detector copies its baseline onto each event so the dashboard can
    // still render the original threshold after ClickHouse rollups expire.
    baselineMean: double("baseline_mean").notNull(),
    baselineStddev: double("baseline_stddev").notNull(),
    thresholdSigma: double("threshold_sigma").notNull(),
    windowStart: bigint("window_start", { mode: "number" }).notNull(),
    windowEnd: bigint("window_end", { mode: "number" }).notNull(),
    ...lifecycleDates,
  },
  (table) => [
    index("status_idx").on(table.status),
    index("workspace_status_fired_at_idx").on(table.workspaceId, table.status, table.firedAt),
    index("workspace_app_environment_status_idx").on(
      table.workspaceId,
      table.appId,
      table.environmentId,
      table.status,
    ),
  ],
);

export const alertEventsRelations = relations(alertEvents, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [alertEvents.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [alertEvents.projectId],
    references: [projects.id],
  }),
  app: one(apps, {
    fields: [alertEvents.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [alertEvents.environmentId],
    references: [environments.id],
  }),
  deployment: one(deployments, {
    fields: [alertEvents.deploymentId],
    references: [deployments.id],
  }),
}));

export type SelectAlertEvent = typeof alertEvents.$inferSelect;
export type InsertAlertEvent = typeof alertEvents.$inferInsert;
