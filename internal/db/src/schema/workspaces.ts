import { relations } from "drizzle-orm";
import {
  datetime,
  int,
  json,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { auditLogs } from "./audit";
import { keys } from "./keys";
import { vercelBindings, vercelIntegrations } from "./vercel_integration";

export const workspaces = mysqlTable(
  "workspaces",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    // Coming from our auth provider clerk
    // This can be either a user_xxx or org_xxx id
    tenantId: varchar("tenant_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }),

    // different plans, this should only be used for visualisations in the ui
    plan: mysqlEnum("plan", ["free", "pro", "enterprise"]).default("free"),

    // stripe
    stripeCustomerId: varchar("stripe_customer_id", { length: 256 }),
    stripeSubscriptionId: varchar("stripe_subscription_id", { length: 256 }),

    // null means there was no trial
    trialEnds: datetime("trial_ends", { fsp: 3 }),
    // if null, you should fall back to start of month
    billingPeriodStart: datetime("billing_period_start", { fsp: 3 }),
    // if null, you should fall back to end of month
    billingPeriodEnd: datetime("billing_period_end", { fsp: 3 }),

    // quotas and usage
    maxActiveKeys: int("quota_max_active_keys"),
    usageActiveKeys: int("usage_active_keys"),
    maxVerifications: int("quota_max_verifications"),
    usageVerifications: int("usage_verifications"),
    lastUsageUpdate: datetime("last_usage_update", { fsp: 3 }),

    /**
     * feature flags
     *
     * betaFeatures may be toggled by the user for early access
     */
    betaFeatures: json("beta_features")
      .$type<{
        auditLog?: boolean;
      }>()
      .notNull(),
    features: json("features")
      .$type<{
        auditLog?: boolean;
      }>()
      .notNull(),
    subscriptions: json("subscriptions").$type<{
      priceIdActiveKeys?: string,
      priceIdVerifications?: string,
      priceIdSupport?: string,
      priceIdPlan?: string,
    }>()
  },
  (table) => ({
    tenantIdIdx: uniqueIndex("tenant_id_idx").on(table.tenantId),
    slugIdx: uniqueIndex("slug_idx").on(table.slug),
  }),
);

export const workspacesRelations = relations(workspaces, ({ many }) => ({
  apis: many(apis),
  keys: many(keys, {
    relationName: "workspace_key_relation",
  }),
  vercelIntegrations: many(vercelIntegrations, {
    relationName: "vercel_workspace_relation",
  }),
  vercelBindings: many(vercelBindings, {
    relationName: "vercel_key_binding_relation",
  }),
  auditLogs: many(auditLogs),
}));
