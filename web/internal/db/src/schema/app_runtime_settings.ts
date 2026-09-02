import { relations, sql } from "drizzle-orm";
import { int, json, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";

export type Healthcheck = {
  method: "GET" | "POST";
  path: string;
  intervalSeconds: number;
  timeoutSeconds: number;
  failureThreshold: number;
  initialDelaySeconds: number;
};

import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const appRuntimeSettings = mysqlTable(
  "app_runtime_settings",
  {
    pk: primaryKey(),

    workspaceId: id("workspace_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),

    port: int("port").notNull().default(8080),
    // CPU allocation in millicores (1000 millicores = 1 CPU).
    cpuMillicores: int("cpu_millicores").notNull().default(250),
    memoryMib: int("memory_mib").notNull().default(256),
    storageMib: int("storage_mib", { unsigned: true }).notNull().default(0),
    command: json("command").$type<string[]>().notNull().default(sql`('[]')`),

    // null = no healthcheck configured
    healthcheck: json("healthcheck").$type<Healthcheck>(),

    shutdownSignal: mysqlEnum("shutdown_signal", ["SIGTERM", "SIGINT", "SIGQUIT", "SIGKILL"])
      .notNull()
      .default("SIGTERM"),

    // Protocol Frontline uses to proxy to the instance (h2c enables gRPC/Connect)
    upstreamProtocol: mysqlEnum("upstream_protocol", ["http1", "h2c"]).notNull().default("http1"),

    sentinelConfig: longblob("sentinel_config").notNull(),

    // null = scraping disabled; non-null path (e.g. /openapi.yaml) enables scraping
    openapiSpecPath: varchar("openapi_spec_path", { length: 512 }),

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
