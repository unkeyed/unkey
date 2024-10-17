import { relations } from "drizzle-orm";
import { boolean, datetime, json, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";
import { workspaces } from "./workspaces";

export const gateways = mysqlTable("gateways", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull().unique(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
});

export const gatewaysRelations = relations(gateways, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [gateways.workspaceId],
    references: [workspaces.id],
  }),
  deployments: many(gatewayDeployments),
  branches: many(gatewayBranches, {
    relationName: "gateway_branches",
  }),
}));

export const gatewayBranches = mysqlTable("gateway_branches", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull().unique(),
  gatewayId: varchar("gateway_id", { length: 256 })
    .notNull()
    .references(() => gateways.id, { onDelete: "cascade" }),

  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),

  domain: varchar("domain", { length: 256 }).notNull().unique(),

  // the gateway.id of the parent
  // this allows us to do merges and migrations like planetscale
  parentId: varchar("parent_id", { length: 256 }),
  activeDeploymentId: varchar("active_deployment_id", { length: 256 }),
  openapi: text("openapi"),
  isMain: boolean("is_main").notNull().default(false),
});

export const gatewayBranchesRelations = relations(gatewayBranches, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [gatewayBranches.workspaceId],
    references: [workspaces.id],
  }),
  deployments: many(gatewayDeployments),
  parent: one(gatewayBranches, {
    fields: [gatewayBranches.parentId],
    references: [gatewayBranches.id],
  }),
  gateway: one(gateways, {
    fields: [gatewayBranches.gatewayId],
    references: [gateways.id],
    relationName: "gateway_branches",
  }),
}));

export const gatewayDeployments = mysqlTable("gateway_deployments", {
  id: varchar("id", { length: 256 }).primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),

  cloudflareDeploymentId: varchar("cloudflare_deployment_id", { length: 256 }),
  gatewayId: varchar("gateway_id", { length: 256 })
    .notNull()
    .references(() => gateways.id, { onDelete: "cascade" }),

  buildStart: datetime("build_start", { fsp: 3 }),
  buildEnd: datetime("build_end", { fsp: 3 }),

  // the generated and used wrangler.json file, useful for debugging
  wranglerConfig: json("wrangler_config"),
});

export const gatewayDeploymentsRelations = relations(gatewayDeployments, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [gatewayDeployments.workspaceId],
    references: [workspaces.id],
  }),
  gateway: one(gateways, {
    fields: [gatewayDeployments.id],
    references: [gateways.id],
  }),
}));
