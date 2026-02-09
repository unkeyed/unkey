import { relations } from "drizzle-orm";
import { boolean, datetime, mysqlTable, text, varchar } from "drizzle-orm/mysql-core";

/**
 * Better Auth tables - prefixed with `ba_` to avoid conflicts with existing tables.
 * These tables are required by the Better Auth library for authentication,
 * session management, and organization features.
 *
 * Note: Better Auth expects datetime columns for date fields, not bigint timestamps.
 */

export const baUser = mysqlTable("ba_user", {
  id: varchar("id", { length: 36 }).primaryKey(),
  name: varchar("name", { length: 256 }),
  email: varchar("email", { length: 256 }).notNull().unique(),
  emailVerified: boolean("email_verified").notNull().default(false),
  image: text("image"),
  role: varchar("role", { length: 64 }),
  banned: boolean("banned").default(false),
  banReason: text("ban_reason"),
  banExpires: datetime("ban_expires", { mode: "date" }),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: datetime("updated_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date())
    .$onUpdateFn(() => new Date()),
});

export const baSession = mysqlTable("ba_session", {
  id: varchar("id", { length: 36 }).primaryKey(),
  userId: varchar("user_id", { length: 36 }).notNull(),
  token: varchar("token", { length: 256 }).notNull().unique(),
  ipAddress: varchar("ip_address", { length: 64 }),
  userAgent: text("user_agent"),
  expiresAt: datetime("expires_at", { mode: "date" }).notNull(),
  activeOrganizationId: varchar("active_organization_id", { length: 36 }),
  impersonatedBy: varchar("impersonated_by", { length: 36 }),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: datetime("updated_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date())
    .$onUpdateFn(() => new Date()),
});

export const baAccount = mysqlTable("ba_account", {
  id: varchar("id", { length: 36 }).primaryKey(),
  userId: varchar("user_id", { length: 36 }).notNull(),
  accountId: varchar("account_id", { length: 256 }).notNull(),
  providerId: varchar("provider_id", { length: 64 }).notNull(),
  accessToken: text("access_token"),
  refreshToken: text("refresh_token"),
  accessTokenExpiresAt: datetime("access_token_expires_at", { mode: "date" }),
  refreshTokenExpiresAt: datetime("refresh_token_expires_at", { mode: "date" }),
  scope: text("scope"),
  idToken: text("id_token"),
  password: text("password"),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: datetime("updated_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date())
    .$onUpdateFn(() => new Date()),
});

export const baVerification = mysqlTable("ba_verification", {
  id: varchar("id", { length: 36 }).primaryKey(),
  identifier: varchar("identifier", { length: 256 }).notNull(),
  value: text("value").notNull(),
  expiresAt: datetime("expires_at", { mode: "date" }).notNull(),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
  updatedAt: datetime("updated_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date())
    .$onUpdateFn(() => new Date()),
});

export const baOrganization = mysqlTable("ba_organization", {
  id: varchar("id", { length: 36 }).primaryKey(),
  name: varchar("name", { length: 256 }).notNull(),
  slug: varchar("slug", { length: 256 }).notNull().unique(),
  logo: text("logo"),
  metadata: text("metadata"),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
});

export const baMember = mysqlTable("ba_member", {
  id: varchar("id", { length: 36 }).primaryKey(),
  userId: varchar("user_id", { length: 36 }).notNull(),
  organizationId: varchar("organization_id", { length: 36 }).notNull(),
  role: varchar("role", { length: 64 }).notNull(),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
});

export const baInvitation = mysqlTable("ba_invitation", {
  id: varchar("id", { length: 36 }).primaryKey(),
  email: varchar("email", { length: 256 }).notNull(),
  inviterId: varchar("inviter_id", { length: 36 }).notNull(),
  organizationId: varchar("organization_id", { length: 36 }).notNull(),
  role: varchar("role", { length: 64 }).notNull(),
  status: varchar("status", { length: 32 }).notNull().default("pending"),
  expiresAt: datetime("expires_at", { mode: "date" }).notNull(),
  createdAt: datetime("created_at", { mode: "date" })
    .notNull()
    .$defaultFn(() => new Date()),
});

// Relations
export const baUserRelations = relations(baUser, ({ many }) => ({
  sessions: many(baSession),
  accounts: many(baAccount),
  memberships: many(baMember),
  invitations: many(baInvitation),
}));

export const baSessionRelations = relations(baSession, ({ one }) => ({
  user: one(baUser, {
    fields: [baSession.userId],
    references: [baUser.id],
  }),
  activeOrganization: one(baOrganization, {
    fields: [baSession.activeOrganizationId],
    references: [baOrganization.id],
  }),
}));

export const baAccountRelations = relations(baAccount, ({ one }) => ({
  user: one(baUser, {
    fields: [baAccount.userId],
    references: [baUser.id],
  }),
}));

export const baOrganizationRelations = relations(baOrganization, ({ many }) => ({
  members: many(baMember),
  invitations: many(baInvitation),
}));

export const baMemberRelations = relations(baMember, ({ one }) => ({
  user: one(baUser, {
    fields: [baMember.userId],
    references: [baUser.id],
  }),
  organization: one(baOrganization, {
    fields: [baMember.organizationId],
    references: [baOrganization.id],
  }),
}));

export const baInvitationRelations = relations(baInvitation, ({ one }) => ({
  inviter: one(baUser, {
    fields: [baInvitation.inviterId],
    references: [baUser.id],
  }),
  organization: one(baOrganization, {
    fields: [baInvitation.organizationId],
    references: [baOrganization.id],
  }),
}));
