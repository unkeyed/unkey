import { relations } from "drizzle-orm";
import {
  bigint,
  datetime,
  index,
  int,
  mysqlTable,
  text,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { auditLogs } from "./audit";
import { keyAuth } from "./keyAuth";
import { workspaces } from "./workspaces";

export const keys = mysqlTable(
  "keys",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    keyAuthId: varchar("key_auth_id", { length: 256 }).notNull(),
    hash: varchar("hash", { length: 256 }).notNull(),
    start: varchar("start", { length: 256 }).notNull(),

    /**
     * This is the workspace that owns the key.
     */
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    /**
     * For internal keys, this is the workspace that the key is for.
     * The owning workspace is an internal one, defined in env.UNKEY_WORKSPACE_ID
     * However in order to filter and display the keys in the UI, we need to know which user/org
     * the key is for.
     *
     * This field is not used for user keys, only for the internal keys that are used to manage the unkey app itself.
     */
    forWorkspaceId: varchar("for_workspace_id", { length: 256 }),
    name: varchar("name", { length: 256 }),
    ownerId: varchar("owner_id", { length: 256 }),
    meta: text("meta"),
    createdAt: datetime("created_at", { fsp: 3 }).notNull(), // unix milli
    expires: datetime("expires", { fsp: 3 }), // unix,
    /**
     * When a key is revoked, we set this time field to mark it as deleted.
     *
     * All places where we show keys, should filter by this field.
     *
     * `deletedAt == null` means the key is active.
     */
    deletedAt: datetime("deleted_at", { fsp: 3 }),
    /**
     * You can limit the amount of times a key can be verified before it becomes invalid
     */
    remaining: int("remaining_requests"),

    ratelimitType: text("ratelimit_type", { enum: ["consistent", "fast"] }),
    ratelimitLimit: int("ratelimit_limit"), // max size of the bucket
    ratelimitRefillRate: int("ratelimit_refill_rate"), // tokens per interval
    ratelimitRefillInterval: int("ratelimit_refill_interval"), // milliseconds
    totalUses: bigint("total_uses", { mode: "number" }).default(0),
  },
  (table) => ({
    hashIndex: uniqueIndex("hash_idx").on(table.hash),
    keyAuthIdIndex: index("key_auth_id_idx").on(table.keyAuthId),
  }),
);

export const keysRelations = relations(keys, ({ one, many }) => ({
  keyAuth: one(keyAuth, {
    fields: [keys.keyAuthId],
    references: [keyAuth.id],
  }),
  workspace: one(workspaces, {
    fields: [keys.workspaceId],
    references: [workspaces.id],
  }),
  forWorkspace: one(workspaces, {
    fields: [keys.forWorkspaceId],
    references: [workspaces.id],
  }),
  auditLog: many(auditLogs),
}));
