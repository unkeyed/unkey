import { relations, sql } from "drizzle-orm";
import {
  bigint,
  boolean,
  index,
  json,
  mysqlEnum,
  mysqlTable,
  varchar,
} from "drizzle-orm/mysql-core";
import { projects } from "./projects";
import { lifecycleDates, softDelete } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export type LogDrainSource = "runtime" | "request";
export type LogDrainEnvironment = "production" | "preview";
export type LogDrainProvider = "axiom";
export type LogDrainDeliveryMode = "batch" | "stream";

// Per-provider config shape stored in log_drains.config. Provider lives
// in its own column; the JSON blob holds only the provider's non-secret
// settings. Matches the Go side under svc/logdrain/internal/sinks/axiom.Config.
export type LogDrainAxiomConfig = { dataset: string; endpoint?: string };
export type LogDrainConfig = LogDrainAxiomConfig;

export type LogDrainFilters = {
  runtime?: {
    minSeverity?: "debug" | "info" | "warn" | "error";
  };
  request?: {
    statusCodes?: string[];
    excludePaths?: string[];
    includeBodies?: boolean;
  };
};

export const logDrains = mysqlTable(
  "log_drains",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 64 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),

    // NULL means workspace-wide. Non-null scopes the drain to a single project.
    projectId: varchar("project_id", { length: 256 }),

    name: varchar("name", { length: 256 }).notNull(),

    provider: mysqlEnum("provider", ["axiom"]).notNull(),

    // Non-secret provider settings (dataset, endpoint, region). See LogDrainConfig.
    config: json("config").$type<LogDrainConfig>().notNull(),

    sources: json("sources")
      .$type<LogDrainSource[]>()
      .notNull()
      .default(sql`('["runtime","request"]')`),

    environments: json("environments")
      .$type<LogDrainEnvironment[]>()
      .notNull()
      .default(sql`('["production"]')`),

    // Empty array means all apps in scope. Populated array filters to those app IDs.
    apps: json("apps").$type<string[]>().notNull().default(sql`('[]')`),

    filters: json("filters").$type<LogDrainFilters>().notNull().default(sql`('{}')`),

    // 'batch' (CH-tail polling) is the v1 path. 'stream' is reserved for the future Vector
    // aggregator path; no schema change needed when it ships.
    deliveryMode: mysqlEnum("delivery_mode", ["batch", "stream"]).notNull().default("batch"),

    enabled: boolean("enabled").notNull().default(true),

    // Set when the row was provisioned through the marketplace
    // (deploy.extension.install). NULL for pre-marketplace drains created
    // directly via the legacy CRUD path. The runtime joins on this when
    // present so an installation-level uninstall takes the drain offline
    // without an explicit cascade write.
    extensionInstallationId: varchar("extension_installation_id", { length: 128 }),

    ...lifecycleDates,
    ...softDelete,
  },
  (table) => [
    index("log_drains_project_idx").on(table.projectId, table.deletedAt),
    index("log_drains_workspace_idx").on(table.workspaceId, table.deletedAt),
    index("log_drains_extension_installation_idx").on(table.extensionInstallationId),
  ],
);

export const logDrainsRelations = relations(logDrains, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [logDrains.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [logDrains.projectId],
    references: [projects.id],
  }),
}));
