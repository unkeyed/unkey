// db.ts
import { boolean, mysqlTable, uniqueIndex, varchar, mysqlEnum } from "drizzle-orm/mysql-core";
import { relations } from "drizzle-orm";
import { apis } from "./apis";
import { keys } from "./keys";

export const workspaces = mysqlTable(
  "workspaces",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    // Coming from our auth provider clerk
    // This can be either a user_xxx or org_xxx id
    tenantId: varchar("tenant_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(),
    // Internal workspaces are used to manage the unkey app itself
    internal: boolean("internal").notNull().default(false),

    // idk, some kind of feature flag was useful
    // enableBetaFeatures: boolean("enable_beta_features").default(false),

    // different plans, this should only be used for visualisations in the ui
    plan: mysqlEnum("plan", ["free", "pro", "enterprise"]).default("free"),
    stripeCustomerId: varchar("stripe_customer_id", { length: 256 }),
    stripeSubscriptionId: varchar("stripe_subscription_id", { length: 256 }),
  },
  (table) => ({
    tenantIdIdx: uniqueIndex("tenant_id_idx").on(table.tenantId),
    slugIdx: uniqueIndex("slug_idx").on(table.slug),
  }),
);

export const workspacesRelations = relations(workspaces, ({ many }) => ({
  apis: many(apis),
  keys: many(keys),
  // Keys required to authorize against unkey itself.
  internalKeys: many(keys),
}));
