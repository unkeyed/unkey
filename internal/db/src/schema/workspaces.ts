import type { Subscriptions } from "@unkey/billing";
import { relations } from "drizzle-orm";
import { boolean, json, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { certificates } from "./certificates";
import { clickhouseWorkspaceSettings } from "./clickhouse_workspace_settings";
import { gateways } from "./gateways";
import { identities } from "./identity";
import { keyAuth } from "./keyAuth";
import { keys } from "./keys";
import { projects } from "./projects";
import { quotas } from "./quota";
import { ratelimitNamespaces } from "./ratelimit";
import { permissions, roles } from "./rbac";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { vercelBindings, vercelIntegrations } from "./vercel_integration";

export const workspaces = mysqlTable("workspaces", {
  id: varchar("id", { length: 256 }).primaryKey(),

  orgId: varchar("org_id", { length: 256 }).notNull().unique(),
  name: varchar("name", { length: 256 }).notNull(),

  // slug is used for the workspace URL
  slug: varchar("slug", { length: 64 }).notNull().unique(),

  // Deployment platform - which partition this workspace deploys to
  partitionId: varchar("partition_id", { length: 256 }),

  // different plans, this should only be used for visualisations in the ui
  // @deprecated - use tier
  plan: mysqlEnum("plan", ["free", "pro", "enterprise"]).default("free"),
  // replaces plan
  tier: varchar("tier", { length: 256 }).default("Free"),

  // stripe
  stripeCustomerId: varchar("stripe_customer_id", { length: 256 }),
  stripeSubscriptionId: varchar("stripe_subscription_id", { length: 256 }),

  /**
   * feature flags
   *
   * betaFeatures may be toggled by the user for early access
   */
  betaFeatures: json("beta_features")
    .$type<{
      /**
       * Can access /app/authorization pages
       */
      rbac?: boolean;

      identities?: boolean;

      /**
       * Can access /logs
       */
      logsPage?: boolean;

      /**
       * Can access /projects
       */

      deployments?: boolean;
    }>()
    .notNull(),
  features: json("features")
    .$type<{
      /**
       * enable audit log retention by specifiying the number of days
       *
       * undefined should fall back to a default
       */
      auditLogRetentionDays?: number;

      /**
       * enable ratelimit retention by specifiying the number of days
       * undefined, 0 or negative means it's disabled
       */
      ratelimitRetentionDays?: number;

      /**
       * How many custom overrides a workspace may create.
       */
      ratelimitOverrides?: number;

      /**
       * Can access /app/success
       */
      successPage?: boolean;

      ipWhitelist?: boolean;
      webhooks?: boolean;
    }>()
    .notNull(),

  /**
   * deprecated, most customers are on stripe subscriptions instead
   */
  subscriptions: json("subscriptions").$type<Subscriptions>(),
  /**
   * if the workspace is disabled, all API requests will be rejected
   */
  enabled: boolean("enabled").notNull().default(true),
  ...deleteProtection,
  ...lifecycleDatesMigration,
});

export const workspacesRelations = relations(workspaces, ({ many, one }) => ({
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
  roles: many(roles),
  permissions: many(permissions),
  ratelimitNamespaces: many(ratelimitNamespaces),
  keySpaces: many(keyAuth),
  identities: many(identities),
  quotas: one(quotas),
  clickhouseSettings: one(clickhouseWorkspaceSettings),

  projects: many(projects),
  gateways: many(gateways),
  certificates: many(certificates),
}));
