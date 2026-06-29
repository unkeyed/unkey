import { relations } from "drizzle-orm";
import { bigint, boolean, json, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const appBuildSettings = mysqlTable(
  "app_build_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),

    // NULL means "no Dockerfile configured": the deploy worker then builds
    // the app with Railpack instead of a Dockerfile.
    dockerfile: varchar("dockerfile", { length: 500 }),
    dockerContext: varchar("docker_context", { length: 500 }).notNull().default("."),
    // NULL means "let Railpack auto-detect". When set, these override Railpack's
    // detected build/install commands (RAILPACK_BUILD_CMD / RAILPACK_INSTALL_CMD)
    // so monorepos can scope the build to a single app. Ignored for Dockerfile builds.
    buildCommand: varchar("build_command", { length: 1000 }),
    installCommand: varchar("install_command", { length: 1000 }),
    watchPaths: json("watch_paths").notNull().$type<string[]>().default([]),
    autoDeploy: boolean("auto_deploy").notNull().default(true),

    ...lifecycleDates,
  },
  (table) => [uniqueIndex("app_build_settings_app_env_idx").on(table.appId, table.environmentId)],
);

export const appBuildSettingsRelations = relations(appBuildSettings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appBuildSettings.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appBuildSettings.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [appBuildSettings.environmentId],
    references: [environments.id],
  }),
}));
