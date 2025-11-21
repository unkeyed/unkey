import { relations } from "drizzle-orm";
import { index, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";

export const acmeUsers = mysqlTable(
  "acme_users",
  {
    id: varchar("id", { length: 128 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    encryptedKey: text("encrypted_key").notNull(),
    registrationURI: text("registration_uri"),
    ...lifecycleDates,
  },
  (table) => ({
    domainIdx: index("domain_idx").on(table.workspaceId),
  }),
);

export const acmeUsersRelations = relations(acmeUsers, () => ({
  // Relations defined but no foreign keys enforced
  // workspace: one(workspaces),
  // project: one(projects),
  // certificate: one(certificates),
  // route: one(routes),
}));
