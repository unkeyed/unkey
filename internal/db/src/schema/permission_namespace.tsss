import { relations } from "drizzle-orm";
import {
  bigint,
  mysqlEnum,
  mysqlTable,
  timestamp,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { auditLogs } from "./audit";
import { keyAuth } from "./keyAuth";
import { workspaces } from "./workspaces";

export const permissionNamespace = mysqlTable(
  "permission_namespace",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    name: varchar("name", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  },
  (table) => ({
    uniqueNamePerWorkspace: uniqueIndex("unique_name_per_workspace").on(
      table.name,
      table.workspaceId,
    ),
  }),
);

export const permissionNamespaceRelations = relations(permissionNamespace, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [permissionNamespace.workspaceId],
    references: [workspaces.id],
  }),
}));

export const permissionNamespaceBranch = mysqlTable(
  "permission_namespace_branch",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    parentBranchId: varchar("parent_branch_id", { length: 256 }),
    createdAt: timestamp("created_at", { mode: "number" }).notNull(),
    /**
     * The type of object.
     * This is in the user's world.
     *
     * Let's say they're building google drive. They might have a namespace called `file` or `folder`.
     *
     * object types must be unique per namespace.
     */
    objectType: varchar("object_type", { length: 256 }).notNull(),
  },
  (table) => ({
    uniqueObjectTypePerNamespace: uniqueIndex("unique_object_type_per_namespace").on(
      table.objectType,
      table.namespaceId,
    ),
  }),
);

export const permissionNamespaceMergeRequest = mysqlTable(
  "permission_namespace_merge_request",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
    sourceBranchId: varchar("source_branch_id", { length: 256 }).notNull(),
    destinationBranchId: varchar("destination_branch_id", { length: 256 }).notNull(),
    createdAt: timestamp("created_at", { fsp: 3 }).notNull(),
    mergedAt: timestamp("merged_at", { fsp: 3 }),
    cancelledAt: timestamp("cancelled_at", { fsp: 3 }),
    status: mysqlEnum("status", ["open", "merging", "merged", "cancelled"]).notNull(),
  },
  (table) => ({}),
);

export const permissionTuples = mysqlTable(
  "permission_namespace",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
    createdAt: timestamp("created_at", { fsp: 3 }).notNull(),
    deletedAt: timestamp("deleted_at", { fsp: 3 }),

    /**
     * The type of object.
     * This is in the user's world.
     *
     * Let's say they're building google drive. They might have a namespace called `file` or `folder`.
     *
     * object types must be unique per namespace.
     */
    objectType: varchar("object_type", { length: 256 }).notNull(),

    /**
     * The unique id of the object.
     */
    objectId: varchar("object_id", { length: 256 }).notNull(),

    /**
     * The relation
     */
    relation: varchar("relation", { length: 256 }).notNull(),

    subjectNamespace: varchar("subject_namespace", { length: 256 }).notNull(),
    subjectId: varchar("subject_id", { length: 256 }).notNull(),
  },
  (table) => ({
    uniqueObjectTypePerNamespace: uniqueIndex("unique_object_type_per_namespace").on(
      table.objectType,
      table.namespaceId,
    ),
  }),
);

export const permissionTuplesRelations = relations(permissionTuples, ({ one, many }) => ({
  namespace: one(permissionNamespace, {
    fields: [permissionTuples.namespaceId],
    references: [permissionNamespace.id],
  }),
}));

export const permissionNamespaceSchema = mysqlTable(
  "permission_namespace_schema",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    createdAt: timestamp("created_at", { fsp: 3 }).notNull(),
    namespaceId: varchar("namespace_id", { length: 256 }).notNull(),
  },
  (_table) => ({}),
);

export const permissionNamespaceSchemaRelations = relations(
  permissionNamespaceSchema,
  ({ one, many }) => ({
    namespace: one(permissionNamespace, {
      fields: [permissionNamespaceSchema.namespaceId],
      references: [permissionNamespace.id],
    }),
  }),
);
