import { relations, sql } from "drizzle-orm";
import {
  bigint,
  int,
  json,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export type Healthcheck = {
  method: "GET" | "POST";
  path: string;
  intervalSeconds: number;
  timeoutSeconds: number;
  failureThreshold: number;
  initialDelaySeconds: number;
};

export const environmentRuntimeSettings = mysqlTable(
  "environment_runtime_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),

    port: int("port").notNull().default(8080),
    cpuMillicores: int("cpu_millicores").notNull().default(256),
    memoryMib: int("memory_mib").notNull().default(256),
    command: json("command").$type<string[]>().notNull().default(sql`('[]')`),

    // null = no healthcheck configured
    healthcheck: json("healthcheck").$type<Healthcheck>(),

    // Maps region ID to replica count, e.g. {"us-east-1": 3, "eu-central-1": 1}
    // Empty object = 1 replica in all available regions (default behavior)
    regionConfig: json("region_config")
      .$type<Record<string, number>>()
      .notNull()
      .default(sql`('{}')`),

    shutdownSignal: mysqlEnum("shutdown_signal", ["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"])
      .notNull()
      .default("SIGTERM"),

    ...lifecycleDates,
  },
  (table) => [uniqueIndex("env_runtime_settings_environment_id_idx").on(table.environmentId)],
);

export const environmentRuntimeSettingsRelations = relations(
  environmentRuntimeSettings,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [environmentRuntimeSettings.workspaceId],
      references: [workspaces.id],
    }),
    environment: one(environments, {
      fields: [environmentRuntimeSettings.environmentId],
      references: [environments.id],
    }),
  }),
);
