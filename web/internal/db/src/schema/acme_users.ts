import { relations } from "drizzle-orm";
import { index, mysqlTable, text } from "drizzle-orm/mysql-core";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";

export const acmeUsers = mysqlTable(
  "acme_users",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
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
