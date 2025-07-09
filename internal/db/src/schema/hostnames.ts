import { relations } from "drizzle-orm";
import {
  boolean,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { lifecycleDates } from "./util/lifecycle_dates";

export const hostnames = mysqlTable(
  "hostnames",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Domain information
    hostname: varchar("hostname", { length: 256 }).notNull(),

    // Domain type
    isCustomDomain: boolean("is_custom_domain").notNull().default(false),

    // Certificate management
    certificateId: varchar("certificate_id", { length: 256 }),

    // Custom domain verification
    verificationStatus: mysqlEnum("verification_status", [
      "pending",
      "verified",
      "failed",
      "expired",
    ]).default("pending"),

    // Verification tokens/records for domain ownership
    verificationToken: varchar("verification_token", { length: 256 }),
    verificationMethod: mysqlEnum("verification_method", [
      "dns_txt",
      "dns_cname",
      "file_upload",
      "automatic",
    ]),

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
    hostnameIdx: uniqueIndex("hostname_idx").on(table.hostname),
    verificationStatusIdx: index("verification_status_idx").on(table.verificationStatus),
    certificateIdx: index("certificate_idx").on(table.certificateId),
  }),
);

export const hostnamesRelations = relations(hostnames, () => ({
  // Relations defined but no foreign keys enforced
  // workspace: one(workspaces),
  // project: one(projects),
  // certificate: one(certificates),
  // route: one(routes),
}));
