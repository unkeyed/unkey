import { relations } from "drizzle-orm";
import { bigint, int, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const environmentBuildSettings = mysqlTable(
  "environment_build_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),

    dockerfile: varchar("dockerfile", { length: 500 }).notNull().default("Dockerfile"),
    dockerContext: varchar("docker_context", { length: 500 }).notNull().default("."),
    buildCpuMillicores: int("build_cpu_millicores").notNull().default(2000),
    buildMemoryMib: int("build_memory_mib").notNull().default(2048),

    ...lifecycleDates,
  },
  (table) => [uniqueIndex("env_build_settings_environment_id_idx").on(table.environmentId)],
);

export const environmentBuildSettingsRelations = relations(environmentBuildSettings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [environmentBuildSettings.workspaceId],
    references: [workspaces.id],
  }),
  environment: one(environments, {
    fields: [environmentBuildSettings.environmentId],
    references: [environments.id],
  }),
}));
