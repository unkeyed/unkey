import { integer, sqliteTable, text } from "drizzle-orm/sqlite-core";

export const keys = sqliteTable("keys", {
  id: text("id").primaryKey(),
  hash: text("hash").notNull(),
  start: text("start").notNull(),
  name: text("name"),
  meta: text("meta"),
  createdAt: integer("created_at", { mode: "timestamp" }), // unix timestamp
  ownerId: text("ownerId"),
  remaining: integer("remaining_requests"),
});
