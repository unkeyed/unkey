import { relations } from "drizzle-orm";
import { bigint, mysqlEnum, mysqlTable, text, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { environments } from "./environments";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { deleteProtection } from "./util/delete_protection";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const appEnvironmentVariables = mysqlTable(
  "app_environment_variables",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    appId: id("app_id").notNull(),
    environmentId: id("environment_id").notNull(),

    key: caseSensitiveVarchar("key", { length: 256 }).notNull(),

    // Always encrypted via vault (contains keyId, nonce, ciphertext in the blob).
    // TEXT (65,535-byte capacity) so a 16 KiB plaintext cap fits its base64
    // ciphertext (~22 KiB); RSA-4096 keys and typical cert chains overflow a
    // varchar(4096) column.
    value: text("value").notNull(),

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
