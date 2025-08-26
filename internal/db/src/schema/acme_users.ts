import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  mysqlTable,
  varchar,
  text,
} from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";

export const acmeUsers = mysqlTable(
  "acme_users",
  {
    id: bigint("id", { mode: "number", unsigned: true }).primaryKey().autoincrement(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    encryptedKey: text("encrypted_key").notNull(),
    registrationURI: text("registration_uri"),
    ...lifecycleDates,
  },
  (table) => ({
    domainIdx: index("domain_idx").on(table.workspaceId),
  })
);

export const acmeUsersRelations = relations(acmeUsers, () => ({
  // Relations defined but no foreign keys enforced
  // workspace: one(workspaces),
  // project: one(projects),
  // certificate: one(certificates),
  // route: one(routes),
}));
