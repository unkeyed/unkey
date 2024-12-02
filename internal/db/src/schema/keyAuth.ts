import { relations } from "drizzle-orm";
import { bigint, boolean, datetime, int, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const keyAuth = mysqlTable("key_auth", {
  id: varchar("id", { length: 256 }).primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),
  createdAt: datetime("created_at", { mode: "date", fsp: 3 }),
  deletedAt: datetime("deleted_at", { mode: "date", fsp: 3 }),

  ...lifecycleDatesMigration,

  storeEncryptedKeys: boolean("store_encrypted_keys").notNull().default(false),
  defaultPrefix: varchar("default_prefix", { length: 8 }),
  defaultBytes: int("default_bytes").default(16),

  /**
   * How many keys are in this keyspace.
   * This is an approximation, accurate at the `sizeLastUpdatedAt` timestamp.
   * If `sizeLastUpdatedAt` is older than 1 minute, you must revalidate this field
   * by counting all keys and updating it.
   */
  sizeApprox: int("size_approx").notNull().default(0),
  sizeLastUpdatedAt: bigint("size_last_updated_at", { mode: "number" }).notNull().default(0),
});

export const keyAuthRelations = relations(keyAuth, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [keyAuth.workspaceId],
    references: [workspaces.id],
  }),
  api: one(apis, {
    fields: [keyAuth.id],
    references: [apis.keyAuthId],
  }),
  keys: many(keys),
}));
