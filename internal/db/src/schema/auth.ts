import { relations } from "drizzle-orm";
import { datetime, mysqlEnum, mysqlTable, uniqueIndex, varchar } from "drizzle-orm/mysql-core";
import { workspaces } from "./workspaces";

export const users = mysqlTable("users", {
  id: varchar("id", {
    length: 255,
  }).primaryKey(),

  email: varchar("email", { length: 255 }).notNull().unique(),
  firstName: varchar("first_name", { length: 256 }).notNull(),
  lastName: varchar("last_name", { length: 256 }).notNull(),
  profilePictureUrl: varchar("profile_picture_url", { length: 256 }).notNull(),
  createdAt: datetime("created_at", { fsp: 3 }).notNull(),
  updatedAt: datetime("updated_at", { fsp: 3 }).notNull(),
});

export const usersRelations = relations(users, ({ many }) => ({
  sessions: many(sessions, {
    relationName: "user_sessions",
  }),
  memberships: many(memberships, {
    relationName: "user_memberships",
  }),
}));

export const sessions = mysqlTable("sessions", {
  id: varchar("id", {
    length: 255,
  }).primaryKey(),
  userId: varchar("user_id", {
    length: 255,
  }).notNull(),
  expiresAt: datetime("expires_at", { fsp: 3 }).notNull(),
});

export const sessionsRelations = relations(sessions, ({ one }) => ({
  user: one(users, {
    fields: [sessions.userId],
    references: [users.id],
    relationName: "user_sessions",
  }),
}));

export const memberships = mysqlTable(
  "memberships",
  {
    id: varchar("id", {
      length: 255,
    }).primaryKey(),
    role: mysqlEnum("role", ["member", "admin", "owner"]).notNull().default("member"),
    userId: varchar("user_id", {
      length: 255,
    })
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    workspaceId: varchar("workspace_id", {
      length: 255,
    })
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
  },
  (table) => ({
    userWorkspaceUniqueIdx: uniqueIndex("unique_user_id_workspace_id_idx").on(
      table.userId,
      table.workspaceId,
    ),
  }),
);

export const membershipsRelations = relations(memberships, ({ one }) => ({
  user: one(users, {
    fields: [memberships.userId],
    references: [users.id],
    relationName: "user_memberships",
  }),
  workspace: one(workspaces, {
    fields: [memberships.workspaceId],
    references: [workspaces.id],
    relationName: "workspace_memberships",
  }),
}));
