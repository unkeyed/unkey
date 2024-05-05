import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  int,
  mysqlEnum,
  mysqlTable,
  varchar,
} from "drizzle-orm/mysql-core";
import { secrets } from "./secrets";
import { workspaces } from "./workspaces";

export const webhooks = mysqlTable(
  "webhooks",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    destination: varchar("destination", { length: 256 }).notNull(),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    createdAt: bigint("created_at", { mode: "number" })
      .notNull()
      .$defaultFn(() => Date.now()),

    enabled: boolean("enabled").notNull().default(true),

    // authorization secret
    algorithm: mysqlEnum("algorithm", ["AES-GCM"]).notNull(),
    iv: varchar("iv", { length: 255 }).notNull(),
    ciphertext: varchar("ciphertext", { length: 1024 }).notNull(),
    encryptionKeyVersion: int("encryption_key_version").notNull().default(1),
  },
  (table) => ({
    workspaceId: index("workspace_id_idx").on(table.workspaceId),
  }),
);

export const webhooksRelations = relations(webhooks, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [webhooks.workspaceId],
    references: [workspaces.id],
  }),
  secret: one(secrets),
}));
