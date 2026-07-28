import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, text } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";

export const acmeUsers = mysqlTable(
  "acme_users",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: caseSensitiveVarchar("id", { length: 32 }).notNull().unique(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 32 }).notNull(),
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
