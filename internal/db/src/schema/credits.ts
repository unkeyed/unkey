import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  int,
  mysqlTable,
  tinyint,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { identities } from "./identity";
import { keys } from "./keys";
import { lifecycleDatesV2 } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const credits = mysqlTable(
  "credits",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    /**
     * Either keyId or identityId must be defined, not both
     */
    keyId: varchar("key_id", { length: 256 }),
    /**
     * Either keyId or identityId must be defined, not both
     */
    identityId: varchar("identity_id", { length: 256 }),

    /**
     * The number of credits remaining
     */
    remaining: int("remaining").notNull(),

    /**
     * You can refill credits at a desired interval
     *
     * Specify the day on which we should refill.
     * - 1    = we refill on the first of the month
     * - 2    = we refill on the 2nd of the month
     * - 31   = we refill on the 31st or last available day
     * - null = we refill every day
     */
    refillDay: tinyint("refill_day"),
    refillAmount: int("refill_amount"),
    refilledAt: bigint("refilled_at", {
      mode: "number",
      unsigned: true,
    }),
    ...lifecycleDatesV2,
  },
  (table) => ({
    workspaceIdIdx: index("workspace_id_idx").on(table.workspaceId),
    uniquePerKey: uniqueIndex("unique_per_key_idx").on(table.keyId),
    uniquePerIdentity: uniqueIndex("unique_per_identity_idx").on(table.identityId),
    // Index for refill workflow queries
    refillLookupIdx: index("refill_lookup_idx").on(
      table.refillDay,
      table.refillAmount,
      table.refilledAt,
    ),
  }),
);

export const creditsRelations = relations(credits, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [credits.workspaceId],
    references: [workspaces.id],
  }),
  key: one(keys, {
    fields: [credits.keyId],
    references: [keys.id],
  }),
  identity: one(identities, {
    fields: [credits.identityId],
    references: [identities.id],
  }),
}));
