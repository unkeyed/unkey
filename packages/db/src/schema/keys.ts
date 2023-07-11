import {
  mysqlTable,
  varchar,
  json,
  datetime,
  boolean,
  text,
  int,
  uniqueIndex,
} from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { apis } from "./apis";
import { policies } from "./policies";
import { workspaces } from "./workspaces";

export const keys = mysqlTable(
  "keys",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    apiId: varchar("api_id", { length: 256 }).notNull(),
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
     * You can limit the amount of times a key can be verified before it becomes invalid
     */
    remainingRequests: int("remaining_requests"),

    ratelimitType: text("ratelimit_type", { enum: ["consistent", "fast"] }),
    ratelimitLimit: int("ratelimit_limit"), // max size of the bucket
    ratelimitRefillRate: int("ratelimit_refill_rate"), // tokens per interval
    ratelimitRefillInterval: int("ratelimit_refill_interval"), // milliseconds
  },
  (table) => ({
    hashIndex: uniqueIndex("hash_idx").on(table.hash),
  }),
);

export const keysRelations = relations(keys, ({ one, many }) => ({
  api: one(apis, {
    fields: [keys.apiId],
    references: [apis.id],
  }),
  policies: many(policies),
  workspace: one(workspaces, {
    fields: [keys.workspaceId],
    references: [workspaces.id],
  }),
  forWorkspace: one(workspaces, {
    fields: [keys.forWorkspaceId],
    references: [workspaces.id],
  }),
}));
