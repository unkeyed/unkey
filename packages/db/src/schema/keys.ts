import { mysqlTable, varchar, json, datetime, boolean, text, int } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { apis } from "./apis";
import { workspaces } from "./workspaces";
import { policies } from "./policies";

export const keys = mysqlTable("keys", {
  id: varchar("id", { length: 256 }).primaryKey(),
  apiId: varchar("api_id", { length: 256 }),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  hash: varchar("hash", { length: 256 }).notNull(),
  start: varchar("start", { length: 256 }).notNull(),
  ownerId: varchar("owner_id", { length: 256 }),
  meta: json("meta"),
  createdAt: datetime("created_at", { fsp: 3 }).notNull(), // unix milli
  expires: datetime("expires", { fsp: 3 }), // unix
  // Internal keys are used to interact with the unkey API instead of 3rd party users
  internal: boolean("internal"),
  ratelimitType: text("ratelimit_type", { enum: ["consistent", "fast"] }),
  ratelimitLimit: int("ratelimit_limit"), // max size of the bucket
  ratelimitRefillRate: int("ratelimit_refill_rate"), // tokens per interval
  ratelimitRefillInterval: int("ratelimit_refill_interval"), // milliseconds
});

export const keysRelations = relations(keys, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [keys.workspaceId],
    references: [workspaces.id],
  }),
  api: one(apis, {
    fields: [keys.apiId],
    references: [apis.id],
  }),
  policies: many(policies),
}));
