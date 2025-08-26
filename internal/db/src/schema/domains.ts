import { relations } from "drizzle-orm";
import {
  bigint,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";

export const domains = mysqlTable(
  "domains",
  {
    id: varchar("id", { length: 255 }).primaryKey(),

    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),

    projectId: varchar("project_id", { length: 255 }).notNull(),

    // Domain information
    domain: varchar("domain", { length: 255 }).notNull(),

    type: mysqlEnum("type", ["custom", "generated"])
      .notNull()
      .default("generated"),

    // Auto-generated subdomain configuration
    subdomainConfig: json("subdomain_config").$type<{
      // For *.unkey.app subdomains
      prefix?: string; // commit-hash, branch-name, etc.
      type?: "commit" | "branch" | "version" | "custom";
      autoUpdate?: boolean; // Whether to update when new versions are created
    }>(),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    domainIdx: uniqueIndex("domain_idx").on(table.domain),
  })
);

export const domainChallenges = mysqlTable(
  "domain_challenges",
  {
    id: bigint("id", { mode: "number", unsigned: true })
      .primaryKey()
      .autoincrement(),
    workspaceId: varchar("workspace_id", { length: 255 }).notNull(),
    domainId: varchar("domain_id", { length: 255 }).notNull(),
    token: varchar("token", { length: 255 }),
    authorization: varchar("authorization", { length: 255 }),
    // waiting mean's we haven't picked it up yet and it's not started
    status: mysqlEnum("status", [
      "waiting",
      "pending",
      "verified",
      "failed",
      "expired",
    ])
      .notNull()
      .default("pending"),
    type: mysqlEnum("type", ["http-01", "dns-01", "tls-alpn-01"])
      .notNull()
      .default("http-01"),
    ...lifecycleDates,
    expiresAt: bigint("expires_at", {
      mode: "number",
      unsigned: true,
    }),
  },
  (table) => ({
    domainIdWorkspaceIdIdx: uniqueIndex("domainIdWorkspaceId_idx").on(
      table.domainId,
      table.workspaceId
    ),
    domainStatus: index("domain_status_idx").on(table.status),
  })
);

export const domainRelations = relations(domains, () => ({
  // Relations defined but no foreign keys enforced
  // workspace: one(workspaces),
  // project: one(projects),
  // certificate: one(certificates),
  // route: one(routes),
}));
