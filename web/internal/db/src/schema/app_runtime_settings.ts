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
import { apps } from "./apps";
import type { Healthcheck } from "./environment_runtime_settings";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { workspaces } from "./workspaces";

export const appRuntimeSettings = mysqlTable(
  "app_runtime_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
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

    sentinelConfig: longblob("sentinel_config").notNull(),

    ...lifecycleDates,
  },
  (table) => [uniqueIndex("app_runtime_settings_app_env_idx").on(table.appId, table.environmentId)],
);

export const appRuntimeSettingsRelations = relations(appRuntimeSettings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appRuntimeSettings.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appRuntimeSettings.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [appRuntimeSettings.environmentId],
    references: [environments.id],
  }),
}));
