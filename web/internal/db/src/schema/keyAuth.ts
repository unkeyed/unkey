import { relations } from "drizzle-orm";
import { bigint, boolean, index, int, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { keys } from "./keys";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
// import { id } from "./util/id";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const keyAuth = mysqlTable(
  "key_auth",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    // id: id("id").notNull().unique(),
    id: caseSensitiveVarchar("id", { length: 256 }).notNull().unique(),
    // workspaceId: id("workspace_id").notNull(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    // projectId: id("project_id").notNull().default(""),
    projectId: caseSensitiveVarchar("project_id", { length: 64 }).notNull().default(""),

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
  },
  (table) => [index("key_auth_project_id_idx").on(table.projectId)],
);

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
