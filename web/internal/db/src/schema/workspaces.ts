import { relations } from "drizzle-orm";
import { bigint, boolean, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apis } from "./apis";
import { billingSubscriptions } from "./billing_subscriptions";
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
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDatesMigration } from "./util/lifecycle_dates";
import { workspaceBilling } from "./workspace_billing";

export const workspaces = mysqlTable("workspaces", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  id: caseSensitiveVarchar("id", { length: 256 }).notNull().unique(),

  orgId: caseSensitiveVarchar("org_id", { length: 256 }).notNull().unique(),
  name: varchar("name", { length: 256 }).notNull(),

  // slug is used for the workspace URL
  slug: varchar("slug", { length: 64 }).notNull().unique(),

  k8sNamespace: varchar("k8s_namespace", { length: 256 }).unique(),

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
   * Legacy usage-based billing state for the handful of workspaces still on
   * custom JSON pricing that the modern tier model cannot represent. Kept until
   * those customers are migrated to workspace_billing; NULL / {} means the
   * workspace is on the current billing path.
   */
  // biome-ignore lint/suspicious/noExplicitAny: legacy free-form billing blob
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
  billing: one(workspaceBilling),
  billingSubscriptions: many(billingSubscriptions),
  clickhouseSettings: one(clickhouseWorkspaceSettings),

  projects: many(projects),
  certificates: many(certificates),
}));
