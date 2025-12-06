import { relations } from "drizzle-orm";
import { mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { environments } from "./environments";

export const environmentVariables = mysqlTable(
  "environment_variables",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    environmentId: varchar("environment_id", {
      length: 128,
    }).notNull(),

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
  (table) => [uniqueIndex("environment_id_key").on(table.environmentId, table.key)],
);

export const environmentVariablesRelations = relations(environmentVariables, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [environmentVariables.workspaceId],
    references: [workspaces.id],
  }),
  project: one(environments, {
    fields: [environmentVariables.environmentId],
    references: [environments.id],
  }),
}));
