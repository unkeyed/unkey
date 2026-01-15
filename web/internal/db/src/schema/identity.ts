import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  int,
  json,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { keys } from "./keys";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const identities = mysqlTable(
  "identities",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    /**
     * The external id is used to create a reference to the user's existing data.
     * They likely have an organization or user id at hand
     */
    externalId: varchar("external_id", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    environment: varchar("environment", { length: 256 }).notNull().default("default"),
    meta: json("meta").$type<Record<string, unknown>>(),
    deleted: boolean("deleted").notNull().default(false),
    ...lifecycleDates,
  },
  (table) => ({
    uniqueDeletedExternalIdPerWorkspace: uniqueIndex("workspace_id_external_id_deleted_idx").on(
      table.workspaceId,
      table.externalId,
      table.deleted,
    ),
  }),
);

export const identitiesRelations = relations(identities, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [identities.workspaceId],
    references: [workspaces.id],
  }),
  keys: many(keys),
  ratelimits: many(ratelimits),
}));

/**
 * Ratelimits can be attached to a key or identity and will be referenced by the name
 */
export const ratelimits = mysqlTable(
  "ratelimits",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    /**
     * The name is used to reference this limit when verifying a key.
     */
    name: varchar("name", { length: 256 }).notNull(),

    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    ...lifecycleDates,
    /**
     * Either keyId or identityId may be defined, not both
     */
    keyId: varchar("key_id", { length: 256 }),
    /**
     * Either keyId or identityId may be defined, not both
     */
    identityId: varchar("identity_id", { length: 256 }),
    limit: int("limit").notNull(),
    // milliseconds
    duration: bigint("duration", { mode: "number" }).notNull(),

    // if enabled we will use this limit when verifying a key, whether they
    // specified the name in the request or not
    autoApply: boolean("auto_apply").notNull().default(false),
  },
  (table) => [
    uniqueIndex("unique_name_per_key_idx").on(table.keyId, table.name),
    uniqueIndex("unique_name_per_identity_idx").on(table.identityId, table.name),
  ],
);

export const ratelimitRelations = relations(ratelimits, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [ratelimits.workspaceId],
    references: [workspaces.id],
  }),
  keys: one(keys, {
    fields: [ratelimits.keyId],
    references: [keys.id],
  }),
  identities: one(identities, {
    fields: [ratelimits.identityId],
    references: [identities.id],
  }),
}));
