import { relations } from "drizzle-orm";
import { boolean, index, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { hostnames } from "./hostnames";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { versions } from "./versions";
import { workspaces } from "./workspaces";
export const routes = mysqlTable(
  "routes",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // Hostname this route is for
    hostname: varchar("hostname", { length: 256 }).notNull(),

    // Version routing configuration
    versionId: varchar("version_id", { length: 256 }).notNull(),

    // Route status
    isEnabled: boolean("is_enabled").notNull().default(true),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
    hostnameIdx: uniqueIndex("hostname_idx").on(table.hostname),
    versionIdx: index("version_idx").on(table.versionId),
  }),
);

export const routesRelations = relations(routes, ({ one }) => ({
  // Relations defined but no foreign keys enforced
  workspace: one(workspaces),
  project: one(projects),
  version: one(versions),
  hostname: one(hostnames),
}));
