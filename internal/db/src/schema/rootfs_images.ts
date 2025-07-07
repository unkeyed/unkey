import { relations } from "drizzle-orm";
import { bigint, index, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { projects } from "./projects";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

export const rootfsImages = mysqlTable(
  "rootfs_images",
  {
    id: varchar("id", { length: 256 }).primaryKey(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    projectId: varchar("project_id", { length: 256 }).notNull(),

    // S3 storage location
    // S3 key is just the rootfs image ID
    s3Bucket: varchar("s3_bucket", { length: 256 }).notNull(),
    s3Key: varchar("s3_key", { length: 500 }).notNull(),

    // Metadata
    sizeBytes: bigint("size_bytes", { mode: "number" }).notNull(),

    ...lifecycleDates,
  },
  (table) => ({
    workspaceIdx: index("workspace_idx").on(table.workspaceId),
    projectIdx: index("project_idx").on(table.projectId),
  }),
);

export const rootfsImagesRelations = relations(rootfsImages, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [rootfsImages.workspaceId],
    references: [workspaces.id],
  }),
  project: one(projects, {
    fields: [rootfsImages.projectId],
    references: [projects.id],
  }),
  // builds: many(builds),
  // versions: many(versions),
}));
