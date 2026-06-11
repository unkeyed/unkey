import { relations } from "drizzle-orm";
import { bigint, boolean, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { certificates } from "./certificates";
import { clickhouseWorkspaceSettings } from "./clickhouse_workspace_settings";
import { githubAppInstallations } from "./github_app";
import { identities } from "./identity";
import { keyAuth } from "./keyAuth";
import { keys } from "./keys";
import { projects } from "./projects";
import { quotas } from "./quota";
import { ratelimitNamespaces } from "./ratelimit";
import { permissions, roles } from "./rbac";
import { sentinels } from "./sentinels";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";

export const workspaces = mysqlTable("workspaces", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  id: varchar("id", { length: 256 }).notNull().unique(),

  orgId: varchar("org_id", { length: 256 }).notNull().unique(),
  name: varchar("name", { length: 256 }).notNull(),

  // slug is used for the workspace URL
  slug: varchar("slug", { length: 64 }).notNull().unique(),

  k8sNamespace: varchar("k8s_namespace", { length: 256 }).unique(),

  tier: varchar("tier", { length: 256 }).default("Free"),

  // stripe
  stripeCustomerId: varchar("stripe_customer_id", { length: 256 }),
  stripeSubscriptionId: varchar("stripe_subscription_id", { length: 256 }),

  /**
   * Local mirror of the workspace's Unkey Deploy plan, synced from Stripe by the
   * customer.subscription.* webhook. NULL means no Deploy plan (cannot use
   * Deploy). Lets the deploy gate and dashboard read entitlement without calling
   * Stripe in the hot path. Stripe stays source of truth; this is a cache.
   */
  deployPlan: varchar("deploy_plan", { length: 64 }),

  /**
   * Manual Deploy entitlement override for internal / comped workspaces, owned
   * by us and never touched by the Stripe webhook. NULL = no override. When set
   * (to a plan value), the deploy gate treats the workspace as entitled even
   * without a paid deploy_plan. Kept separate from deployPlan so that stays a
   * pure Stripe mirror.
   */
  deployPlanOverride: varchar("deploy_plan_override", { length: 64 }),

  /**
   * Monthly Compute (Deploy) spend budget in USD cents, set by workspace
   * admins in the dashboard. NULL = no budget. Email alerts fire at fixed
   * percentages of the budget (50/75/100); deploySpendBudgetStop additionally
   * stops workloads when month-to-date usage spend reaches it. v1 stores the
   * preferences only: nothing alerts or stops yet.
   */
  deploySpendBudgetCents: bigint("deploy_spend_budget_cents", {
    mode: "number",
    unsigned: true,
  }),
  deploySpendBudgetStop: boolean("deploy_spend_budget_stop").notNull().default(false),

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
       * Can access customer billing portal
       */
      portal?: boolean;
    }>()
    .notNull(),
  /**
   * deprecated, most customers are on stripe subscriptions instead
   */
  // biome-ignore lint/suspicious/noExplicitAny: legacy field, will be removed
  subscriptions: json("subscriptions").$type<any>(),
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
  roles: many(roles),
  permissions: many(permissions),
  ratelimitNamespaces: many(ratelimitNamespaces),
  keySpaces: many(keyAuth),
  identities: many(identities),
  githubAppInstallations: many(githubAppInstallations),
  quotas: one(quotas),
  clickhouseSettings: one(clickhouseWorkspaceSettings),

  projects: many(projects),
  sentinels: many(sentinels),
  certificates: many(certificates),
}));
