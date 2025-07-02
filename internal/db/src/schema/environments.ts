import { relations } from "drizzle-orm";
import { boolean, index, mysqlTable, text, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const environments = mysqlTable(
  "environments",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(), // production, preview, staging, etc.
    description: text("description"),

    // Whether this is a default environment (production, preview)
    isDefault: boolean("is_default").notNull().default(false),

    ...deleteProtection,
    ...lifecycleDatesMigration,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    workspaceNameIdx: uniqueIndex("workspace_name_idx").on(table.workspaceId, table.name),
  }),
);

export const environmentsRelations = relations(environments, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [environments.workspaceId],
    references: [workspaces.id],
  }),
  // branches: many(branches),
  // versions: many(versions),
  // envVariables: many(envVariables),
}));
