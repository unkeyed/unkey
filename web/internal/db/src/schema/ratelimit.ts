import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  int,
  mysqlEnum,
  mysqlTable,
  unique,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const ratelimitNamespaces = mysqlTable(
  "ratelimit_namespaces",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 512 }).notNull(),

    ...lifecycleDatesMigration,
  },
  (table) => {
    return {
      uniqueNamePerWorkspaceIdx: unique("unique_name_per_workspace_idx").on(
        table.workspaceId,
        table.name,
      ),
    };
  },
);

export const ratelimitNamespaceRelations = relations(ratelimitNamespaces, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [ratelimitNamespaces.workspaceId],
    references: [workspaces.id],
  }),
  overrides: many(ratelimitOverrides),
}));

export const ratelimitOverrides = mysqlTable(
  "ratelimit_overrides",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
    identifier: varchar("identifier", { length: 512 }).notNull(),

    limit: int("limit").notNull(),
    /**
     * window duration in milliseconds
     */
    duration: int("duration").notNull(),
    /**
     * If true, don't wait for the origin to return, use cached values instead.
     */
    async: boolean("async"),

    /**
     * Sharding method used.
     *
     * - edge: use the worker's edge location as part of the DO id, to run local objects
     */
    sharding: mysqlEnum("sharding", ["edge"]),

    ...lifecycleDatesMigration,
  },
  (table) => {
    return {
      uniqueIdentifierPerNamespace: unique("unique_identifier_per_namespace_idx").on(
        table.namespaceId,
        table.identifier,
      ),
    };
  },
);
export const ratelimitOverridesRelations = relations(ratelimitOverrides, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [ratelimitOverrides.workspaceId],
    references: [workspaces.id],
  }),
  namespace: one(ratelimitNamespaces, {
    fields: [ratelimitOverrides.namespaceId],
    references: [ratelimitNamespaces.id],
  }),
}));
