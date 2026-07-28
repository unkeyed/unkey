import { relations } from "drizzle-orm";
import { bigint, boolean, json, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const appBuildSettings = mysqlTable(
  "app_build_settings",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 64 }).notNull(),
    environmentId: caseSensitiveVarchar("environment_id", { length: 128 }).notNull(),

    // NULL means "no Dockerfile configured": the deploy worker then builds
    // the app with Railpack instead of a Dockerfile.
    dockerfile: caseSensitiveVarchar("dockerfile", { length: 500 }),
    dockerContext: caseSensitiveVarchar("docker_context", { length: 500 }).notNull().default("."),
    // NULL means "let Railpack auto-detect". When set, it overrides Railpack's
    // detected build command (RAILPACK_BUILD_CMD) so monorepos can scope the
    // build to a single app. Ignored for Dockerfile builds.
    buildCommand: caseSensitiveVarchar("build_command", { length: 1000 }),
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
