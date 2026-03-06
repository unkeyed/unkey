import { relations } from "drizzle-orm";
import { bigint, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const appEnvironmentVariables = mysqlTable(
  "app_environment_variables",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 128 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    appId: varchar("app_id", { length: 64 }).notNull(),
    environmentId: varchar("environment_id", { length: 128 }).notNull(),

    key: varchar("key", { length: 256 }).notNull(),

    // Always encrypted via vault (contains keyId, nonce, ciphertext in the blob)
    value: varchar("value", { length: 4096 }).notNull(),

    // Both types are encrypted in the database
    // - recoverable: can be decrypted and shown in the UI
    // - writeonly: cannot be read back after creation
    type: mysqlEnum("type", ["recoverable", "writeonly"]).notNull(),

    description: varchar("description", { length: 255 }),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [uniqueIndex("app_env_id_key").on(table.appId, table.environmentId, table.key)],
);

export const appEnvironmentVariablesRelations = relations(appEnvironmentVariables, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appEnvironmentVariables.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appEnvironmentVariables.appId],
    references: [apps.id],
  }),
  environment: one(environments, {
    fields: [appEnvironmentVariables.environmentId],
    references: [environments.id],
  }),
}));
