import { mysqlTable, varchar, primaryKey } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { keys } from "./keys";
import { policies } from "./policies";

export const keysToPolicies = mysqlTable(
  "keys_to_policies",
  {
    keyId: varchar("key_id", { length: 256 }).notNull(),
    policyId: varchar("policy_id", { length: 256 }).notNull(),
  },
  (t) => ({
    pk: primaryKey(t.keyId, t.policyId),
  }),
);

export const keysToPoliciesRelations = relations(keysToPolicies, ({ one }) => ({
  key: one(policies, {
    fields: [keysToPolicies.keyId],
    references: [policies.id],
  }),
  policy: one(keys, {
    fields: [keysToPolicies.policyId],
    references: [keys.id],
  }),
}));
