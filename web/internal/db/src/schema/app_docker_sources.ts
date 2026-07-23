import { relations } from "drizzle-orm";
import { bigint, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * The docker-image counterpart of `github_repo_connections`: for apps with
 * `source_type = 'docker_image'` this stores the default image reference
 * deployed when a deployment is created without an explicit image.
 *
 * Kept as its own table (rather than a column on `apps`) because the source
 * will grow fields that only make sense for image apps: private registry
 * credentials, tag-update policy, digest pinning.
 */
export const appDockerSources = mysqlTable("app_docker_sources", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
  appId: varchar("app_id", { length: 64 }).notNull().unique(),
  // Full image reference including registry, repository, and tag or digest,
  // e.g. `ghcr.io/acme/mcp-server:v1.2.3` or `nginx@sha256:...`.
  image: varchar("image", { length: 512 }).notNull(),
  ...lifecycleDates,
});

export const appDockerSourcesRelations = relations(appDockerSources, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appDockerSources.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appDockerSources.appId],
    references: [apps.id],
  }),
}));
