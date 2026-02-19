import { relations } from "drizzle-orm";
import { bigint, int, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const appInternalServices = mysqlTable(
  "app_internal_services",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),
    region: varchar("region", { length: 64 }).notNull(),
    k8sServiceName: varchar("k8s_service_name", { length: 255 }).notNull(),
    k8sNamespace: varchar("k8s_namespace", { length: 255 }).notNull(),
    port: int("port").notNull(),

    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("one_app_svc_per_env_per_region").on(
      table.appId,
      table.environmentId,
      table.region,
    ),
  ],
);

export const appInternalServicesRelations = relations(appInternalServices, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appInternalServices.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appInternalServices.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [appInternalServices.environmentId],
    references: [environments.id],
  }),
}));
