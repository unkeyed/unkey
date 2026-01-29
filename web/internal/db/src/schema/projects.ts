import { relations, sql } from "drizzle-orm";
import { bigint, boolean, json, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { deleteProtection } from "./util/delete_protection";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

import { deployments } from "./deployments";
import { frontlineRoutes } from "./frontline_routes";
export const projects = mysqlTable(
  "projects",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    name: varchar("name", { length: 256 }).notNull(),
    slug: varchar("slug", { length: 256 }).notNull(), // URL-safe identifier within workspace

    // Git configuration
    gitRepositoryUrl: varchar("git_repository_url", { length: 500 }),
    // this is likely temporary but we need a way to point to the current prod deployment.
    // in the future I think we want to have a special deployment per environment, but for now this is fine
    liveDeploymentId: varchar("live_deployment_id", { length: 256 }),
    isRolledBack: boolean("is_rolled_back").notNull().default(false),
    defaultBranch: varchar("default_branch", { length: 256 }).default("main"),
    depotProjectId: varchar("depot_project_id", { length: 255 }),

    // Unique CNAME target for custom domains (e.g., "k3n5p8x2.unkey-dns.com")
    // Users point their custom domain CNAME to this target
    cnameTarget: varchar("cname_target", { length: 128 }),

    // Default container command override for deployments (e.g., ["./app", "serve"])
    // If empty, the container's default entrypoint/cmd is used
    command: json("command").$type<string[]>().notNull().default(sql`('[]')`),

    ...deleteProtection,
    ...lifecycleDates,
  },
  (table) => [uniqueIndex("workspace_slug_idx").on(table.workspaceId, table.slug)],
);

export const projectsRelations = relations(projects, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [projects.workspaceId],
    references: [workspaces.id],
  }),
  deployments: many(deployments),
  activeDeployment: one(deployments, {
    fields: [projects.liveDeploymentId],
    references: [deployments.id],
  }),
  frontlineRoutes: many(frontlineRoutes),
  // environments: many(projectEnvironments),
}));
