import { relations } from "drizzle-orm";
import { mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { apps } from "./apps";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

export const appSourceOci = mysqlTable(
  "app_source_oci",
  {
    pk: primaryKey(),
    workspaceId: id("workspace_id").notNull(),
    appId: id("app_id").notNull(),
    imageReference: varchar("image_reference", { length: 512 }).notNull(),
    ...lifecycleDates,
  },
  (table) => [uniqueIndex("app_source_oci_app_id_idx").on(table.appId)],
);

export const appSourceOciRelations = relations(appSourceOci, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [appSourceOci.workspaceId],
    references: [workspaces.id],
  }),
  app: one(apps, {
    fields: [appSourceOci.appId],
    references: [apps.id],
  }),
}));
