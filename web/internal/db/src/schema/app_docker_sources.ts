import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const appDockerSources = mysqlTable(
  "app_docker_sources",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    workspaceId: caseSensitiveVarchar("workspace_id", { length: 256 }).notNull(),
    appId: caseSensitiveVarchar("app_id", { length: 64 }).notNull(),
    imageReference: varchar("image_reference", { length: 512 }).notNull(),
    ...lifecycleDates,
  },
  (table) => [uniqueIndex("app_docker_sources_app_id_idx").on(table.appId)],
);

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
