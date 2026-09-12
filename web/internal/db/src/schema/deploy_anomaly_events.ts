import { bigint, index, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

export const deployAnomalyEvents = mysqlTable(
  "deploy_anomaly_events",
  {
    pk: primaryKey(),
    id: varchar("id", { length: 64 }).notNull(),
    workspaceId: id("workspace_id").notNull(),
    projectId: id("project_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),
    deploymentId: id("deployment_id").notNull(),
    metric: mysqlEnum("metric", ["oom_killed", "crash_loop"]).notNull(),
    eventTime: bigint("event_time", { mode: "number" }).notNull(),
    receivedAt: bigint("received_at", { mode: "number" }).notNull(),
    processedAt: bigint("processed_at", { mode: "number" }),
  },
  (table) => [
    uniqueIndex("deploy_anomaly_events_id_unique").on(table.id),
    index("deploy_anomaly_events_pending_idx").on(table.workspaceId, table.processedAt, table.pk),
  ],
);
