import { relations } from "drizzle-orm";
import { bigint, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { environments } from "./environments";
export const environmentVariables = mysqlTable(
  "environment_variables",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    environmentId: bigint("environment_id", {
      mode: "number",
      unsigned: true,
    }).notNull(),

    key: varchar("key", { length: 256 }).notNull(),
    // Either the plaintext value or a vault encrypted response
    value: varchar("value", { length: 1024 }).notNull(),
    type: mysqlEnum("type", ["plaintext", "secret"]).notNull(),

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
