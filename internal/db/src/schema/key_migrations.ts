import { relations } from "drizzle-orm";
import { json, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { keyAuth } from "./keyAuth";
import { workspaces } from "./workspaces";

export const keyMigrations = mysqlTable("key_migrations", {
  id: varchar("id", { length: 256 }).primaryKey(),
  keyAuthId: varchar("key_auth_id", { length: 256 })
    .notNull()
    .references(() => keyAuth.id, { onDelete: "cascade" }),

  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
});

export const keyMigrationsRelations = relations(keyMigrations, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [keyMigrations.workspaceId],
    references: [workspaces.id],
  }),
  errors: many(keyMigrationErrors),
}));

export const keyMigrationErrors = mysqlTable("key_migration_errors", {
  id: varchar("id", { length: 256 }).primaryKey(),
  migrationId: varchar("migration_id", { length: 256 })
    .notNull()
    .references(() => keyMigrations.id, { onDelete: "cascade" }),

  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),

  request: json("request").notNull(),
  error: text("error").notNull(),
});

export const keyMigrationErrorsRelations = relations(keyMigrationErrors, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [keyMigrationErrors.workspaceId],
    references: [workspaces.id],
  }),
  migration: one(keyMigrations, {
    fields: [keyMigrationErrors.workspaceId],
    references: [keyMigrations.id],
  }),
}));
