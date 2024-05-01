import { relations } from "drizzle-orm";
import { datetime, index, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { secrets } from "./secrets";
import { workspaces } from "./workspaces";

export const gateways = mysqlTable(
  "gateways",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    name: varchar("name", { length: 128 }).unique().notNull(),
    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),

    origin: varchar("origin", { length: 256 }).notNull(),
    createdAt: datetime("created_at", { mode: "date", fsp: 3 }).$default(() => new Date()),
    updatedAt: datetime("updated_at", { mode: "date", fsp: 3 }),
    deletedAt: datetime("deleted_at", { mode: "date", fsp: 3 }),
  },
  (table) => ({
    workspaceId: index("workspace_id_idx").on(table.workspaceId),
  }),
);

export const gatewaysRelations = relations(gateways, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [gateways.workspaceId],
    references: [workspaces.id],
  }),
  headerRewrites: many(gatewayHeaderRewrites),
  deployments: many(gatewayDeployments),
  branches: many(gatewayBranches, {
    relationName: "gateway_branches",
  }),
}));

export const gatewayHeaderRewrites = mysqlTable(
  "gateway_header_rewrites",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    name: varchar("name", { length: 256 }).notNull(),
    secretId: varchar("secret_id", { length: 256 }).notNull(),

    workspaceId: varchar("workspace_id", { length: 256 })
      .notNull()
      .references(() => workspaces.id, { onDelete: "cascade" }),
    gatewayId: varchar("gateways_id", { length: 256 })
      .notNull()
      .references(() => gateways.id, { onDelete: "cascade" }),

    createdAt: datetime("created_at", { mode: "date", fsp: 3 }).$default(() => new Date()),
    deletedAt: datetime("deleted_at", { mode: "date", fsp: 3 }),
  },
  (table) => ({
    workspaceId: index("workspace_id_idx").on(table.workspaceId),
  }),
);

export const gatewayHeaderRewritesRelations = relations(gatewayHeaderRewrites, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [gatewayHeaderRewrites.workspaceId],
    references: [workspaces.id],
  }),
  proxy: one(gateways, {
    fields: [gatewayHeaderRewrites.gatewayId],
    references: [gateways.id],
  }),
  secret: one(secrets, {
    fields: [gatewayHeaderRewrites.secretId],
    references: [secrets.id],
  }),
}));

export const gatewayBranches = mysqlTable("gateway_branches", {
  id: varchar("id", { length: 256 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull().unique(),
  createdAt: datetime("created_at", { fsp: 3 }).$default(() => new Date()),
  updatedAt: datetime("updated_at", { fsp: 3 }),
  deletedAt: datetime("deleted_at", { fsp: 3 }),

  gatewayId: varchar("gateway_id", { length: 256 })
    .notNull()
    .references(() => gateways.id, { onDelete: "cascade" }),

  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),

  origin: varchar("origin", { length: 256 }).notNull(),
  domain: varchar("domain", { length: 256 }).notNull().unique(),

  // the gateway.id of the parent
  // this allows us to do merges and migrations like planetscale
  parentId: varchar("parent_id", { length: 256 }),
  activeDeploymentId: varchar("active_deployment_id", { length: 256 }),
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
  // publicly visible
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

export const gatewayHostNames = mysqlTable("gateway_host_names", {
  id: varchar("id", { length: 256 }).primaryKey(),

  hostname: varchar("host_name", { length: 256 }).notNull(),
  workspaceId: varchar("workspace_id", { length: 256 })
    .notNull()
    .references(() => workspaces.id, { onDelete: "cascade" }),

  deploymentId: varchar("deployment_id", { length: 256 }).notNull(),
  cloudflareHostId: varchar("cloudflare_host_id", { length: 256 }).notNull(),
  gatewayId: varchar("gateway_id", { length: 256 })
    .notNull()
    .references(() => gateways.id, { onDelete: "cascade" }),
});

export const gatewayHostNamesRelations = relations(gatewayHostNames, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [gatewayHostNames.workspaceId],
    references: [workspaces.id],
  }),
  gateway: one(gateways, {
    fields: [gatewayHostNames.gatewayId],
    references: [gateways.id],
  }),
}));
