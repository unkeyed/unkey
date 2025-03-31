import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  json,
  mysqlTable,
  primaryKey,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { newId } from "@unkey/id";
import { deleteProtection } from "./util/delete_protection";

export const auditLogBucket = mysqlTable(
  "audit_log_bucket",
  {
    id: varchar("id", { length: 256 })
      .primaryKey()
      .$defaultFn(() => newId("auditLogBucket")),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    /**
     * Buckets are used as namespaces for different logs belonging to a single workspace
     */
    name: varchar("name", { length: 256 }).notNull(),
    /**
     * null means we don't automatically remove logs
     */
    retentionDays: int("retention_days"),
    ...lifecycleDates,
    ...deleteProtection,
  },
  (table) => ({
    uniqueNamePerWorkspace: uniqueIndex("unique_name_per_workspace_idx").on(
      table.workspaceId,
      table.name,
    ),
  }),
);

export const auditLog = mysqlTable(
  "audit_log",
  {
    id: varchar("id", { length: 256 })
      .primaryKey()
      .$defaultFn(() => newId("auditLog")),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // bucket is the name of the bucket that the target belongs to
    bucket: varchar("bucket", { length: 256 }).notNull().default("unkey_mutations"),
    // @deprecated
    bucketId: varchar("bucket_id", { length: 256 }).notNull(),
    event: varchar("event", { length: 256 }).notNull(),

    // When the event happened
    time: bigint("time", { mode: "number" })
      .notNull()
      .$defaultFn(() => Date.now()),
    // A human readable description of the event
    display: varchar("display", { length: 256 }).notNull(),

    remoteIp: varchar("remote_ip", { length: 256 }),
    userAgent: varchar("user_agent", { length: 256 }),
    actorType: varchar("actor_type", { length: 256 }).notNull(),
    actorId: varchar("actor_id", { length: 256 }).notNull(),
    actorName: varchar("actor_name", { length: 256 }),
    actorMeta: json("actor_meta"),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceId: index("workspace_id_idx").on(table.workspaceId),
    bucketId: index("bucket_id_idx").on(table.bucketId),
    bucket: index("bucket_idx").on(table.bucket),
    event: index("event_idx").on(table.event),
    actorId: index("actor_id_idx").on(table.actorId),
    time: index("time_idx").on(table.time),
  }),
);

export const auditLogRelations = relations(auditLog, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [auditLog.workspaceId],
    references: [workspaces.id],
  }),
  // bucket: one(auditLogBucket, {
  //   fields: [auditLog.bucketId],
  //   references: [auditLogBucket.id],
  // }),
  targets: many(auditLogTarget),
}));

export const auditLogTarget = mysqlTable(
  "audit_log_target",
  {
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    // @deprecated
    bucketId: varchar("bucket_id", { length: 256 }).notNull(),

    // bucket is the name of the bucket that the target belongs to
    bucket: varchar("bucket", { length: 256 }).notNull().default("unkey_mutations"),
    auditLogId: varchar("audit_log_id", { length: 256 }).notNull(),

    // A human readable name to display in the UI
    displayName: varchar("display_name", { length: 256 }).notNull(),
    // the type of the target
    type: varchar("type", { length: 256 }).notNull(),
    // the id of the target
    id: varchar("id", { length: 256 }).notNull(),
    // the name of the target
    name: varchar("name", { length: 256 }),
    // the metadata of the target
    meta: json("meta"),

    ...lifecycleDates,
  },
  (table) => ({
    pk: primaryKey({ columns: [table.auditLogId, table.id] }),
    bucket: index("bucket").on(table.bucket),
    auditLog: index("audit_log_id").on(table.auditLogId),
    id: index("id_idx").on(table.id),
  }),
);

export const auditLogTargetRelations = relations(auditLogTarget, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [auditLogTarget.workspaceId],
    references: [workspaces.id],
  }),
  // bucket: one(auditLogBucket, {
  //   fields: [auditLogTarget.bucketId],
  //   references: [auditLogBucket.id],
  // }),
  log: one(auditLog, {
    fields: [auditLogTarget.auditLogId],
    references: [auditLog.id],
  }),
}));

export type SelectAuditLog = typeof auditLog.$inferSelect;
export type SelectAuditLogTarget = typeof auditLogTarget.$inferSelect;
