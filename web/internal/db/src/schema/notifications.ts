import { relations, sql } from "drizzle-orm";
import {
  bigint,
  boolean,
  check,
  index,
  int,
  json,
  mediumtext,
  mysqlEnum,
  mysqlTable,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import type { Audience } from "../gen/notification/v1/config_pb";
import { apps } from "./apps";
import { environments } from "./environments";
import { projects } from "./projects";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { lifecycleDates } from "./util/lifecycle_dates";
import { longblob } from "./util/longblob";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

const notificationChannels = ["email", "discord", "slack", "webhook"] as const;

export const notificationDestinations = mysqlTable(
  "notification_destinations",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    name: varchar("name", { length: 128 }).notNull(),
    channel: mysqlEnum("channel", ["discord", "slack", "webhook"]).notNull(),
    config: longblob("config").$type<Buffer>().notNull(),
    enabled: boolean("enabled").notNull().default(true),
    ...lifecycleDates,
  },
  (table) => [index("notification_destinations_workspace_idx").on(table.workspaceId)],
);

export const notificationSubscriptions = mysqlTable(
  "notification_subscriptions",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    name: varchar("name", { length: 128 }).notNull(),
    notificationType: caseSensitiveVarchar("notification_type", { length: 128 }).notNull(),
    projectId: id("project_id"),
    appId: id("app_id"),
    environmentId: id("environment_id"),
    enabled: boolean("enabled").notNull().default(true),
    ...lifecycleDates,
  },
  (table) => [
    index("notification_subscriptions_routing_idx").on(
      table.workspaceId,
      table.notificationType,
      table.enabled,
    ),
  ],
);

export const notificationSubscriptionTargets = mysqlTable(
  "notification_subscription_targets",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    subscriptionId: id("subscription_id").notNull(),
    channel: mysqlEnum("channel", notificationChannels).notNull(),
    audience: int("audience", { unsigned: true }).$type<Audience>(),
    userId: caseSensitiveVarchar("user_id", { length: 256 }),
    destinationId: id("destination_id"),
    ...lifecycleDates,
  },
  (table) => [
    index("notification_targets_subscription_idx").on(table.workspaceId, table.subscriptionId),
    index("notification_targets_destination_idx").on(table.workspaceId, table.destinationId),
    check(
      "notification_targets_exactly_one_recipient",
      sql`(${table.audience} IS NOT NULL) + (${table.userId} IS NOT NULL) + (${table.destinationId} IS NOT NULL) = 1`,
    ),
    check(
      "notification_targets_audience_specified",
      sql`${table.audience} IS NULL OR ${table.audience} > 0`,
    ),
    check(
      "notification_targets_channel_recipient",
      sql`(${table.channel} = 'email' AND ${table.destinationId} IS NULL) OR (${table.channel} <> 'email' AND ${table.destinationId} IS NOT NULL)`,
    ),
  ],
);

export const notificationPreferences = mysqlTable(
  "notification_preferences",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    userId: caseSensitiveVarchar("user_id", { length: 256 }).notNull(),
    notificationType: caseSensitiveVarchar("notification_type", { length: 128 }).notNull(),
    channel: mysqlEnum("channel", ["email"]).notNull(),
    enabled: boolean("enabled").notNull(),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("notification_preferences_user_type_channel_idx").on(
      table.workspaceId,
      table.userId,
      table.notificationType,
      table.channel,
    ),
  ],
);

export const notifications = mysqlTable(
  "notifications",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    sourceEventId: caseSensitiveVarchar("source_event_id", { length: 256 }).notNull(),
    notificationType: caseSensitiveVarchar("notification_type", { length: 128 }).notNull(),
    requestHash: caseSensitiveVarchar("request_hash", { length: 64 }).notNull(),
    schemaVersion: int("schema_version", { unsigned: true }).notNull(),
    resourceContext: json("resource_context")
      .$type<{ projectId?: string; appId?: string; environmentId?: string }>()
      .notNull(),
    payload: json("payload").$type<Record<string, unknown>>().notNull(),
    occurredAt: bigint("occurred_at", { mode: "number" }).notNull(),
    expiresAt: bigint("expires_at", { mode: "number" }),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("notifications_source_event_idx").on(
      table.workspaceId,
      table.notificationType,
      table.sourceEventId,
    ),
    index("notifications_workspace_created_idx").on(table.workspaceId, table.createdAt, table.id),
  ],
);

export const notificationDeliveries = mysqlTable(
  "notification_deliveries",
  {
    pk: primaryKey(),
    id: id("id").notNull().unique(),
    workspaceId: id("workspace_id").notNull(),
    notificationId: id("notification_id").notNull(),
    channel: mysqlEnum("channel", notificationChannels).notNull(),
    targetKey: caseSensitiveVarchar("target_key", { length: 320 }).notNull(),
    encryptedTargetSnapshot: mediumtext("encrypted_target_snapshot").notNull(),
    provider: caseSensitiveVarchar("provider", { length: 64 }),
    providerMessageId: caseSensitiveVarchar("provider_message_id", { length: 256 }),
    status: mysqlEnum("status", [
      "pending",
      "sending",
      "accepted",
      "delivered",
      "failed",
      "suppressed",
      "unknown",
    ])
      .notNull()
      .default("pending"),
    reason: varchar("reason", { length: 256 }),
    attemptCount: int("attempt_count", { unsigned: true }).notNull().default(0),
    firstAttemptAt: bigint("first_attempt_at", { mode: "number" }),
    nextAttemptAt: bigint("next_attempt_at", { mode: "number" }),
    ...lifecycleDates,
  },
  (table) => [
    uniqueIndex("notification_deliveries_target_idx").on(
      table.notificationId,
      table.channel,
      table.targetKey,
    ),
    index("notification_deliveries_workspace_notification_idx").on(
      table.workspaceId,
      table.notificationId,
    ),
    index("notification_deliveries_provider_message_idx").on(
      table.provider,
      table.providerMessageId,
    ),
  ],
);

export const notificationDestinationsRelations = relations(
  notificationDestinations,
  ({ one, many }) => ({
    workspace: one(workspaces, {
      fields: [notificationDestinations.workspaceId],
      references: [workspaces.id],
    }),
    subscriptionTargets: many(notificationSubscriptionTargets),
  }),
);

export const notificationSubscriptionsRelations = relations(
  notificationSubscriptions,
  ({ one, many }) => ({
    workspace: one(workspaces, {
      fields: [notificationSubscriptions.workspaceId],
      references: [workspaces.id],
    }),
    project: one(projects, {
      fields: [notificationSubscriptions.workspaceId, notificationSubscriptions.projectId],
      references: [projects.workspaceId, projects.id],
    }),
    app: one(apps, {
      fields: [notificationSubscriptions.workspaceId, notificationSubscriptions.appId],
      references: [apps.workspaceId, apps.id],
    }),
    environment: one(environments, {
      fields: [notificationSubscriptions.workspaceId, notificationSubscriptions.environmentId],
      references: [environments.workspaceId, environments.id],
    }),
    targets: many(notificationSubscriptionTargets),
  }),
);

export const notificationSubscriptionTargetsRelations = relations(
  notificationSubscriptionTargets,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [notificationSubscriptionTargets.workspaceId],
      references: [workspaces.id],
    }),
    subscription: one(notificationSubscriptions, {
      fields: [
        notificationSubscriptionTargets.workspaceId,
        notificationSubscriptionTargets.subscriptionId,
      ],
      references: [notificationSubscriptions.workspaceId, notificationSubscriptions.id],
    }),
    destination: one(notificationDestinations, {
      fields: [
        notificationSubscriptionTargets.workspaceId,
        notificationSubscriptionTargets.destinationId,
        notificationSubscriptionTargets.channel,
      ],
      references: [
        notificationDestinations.workspaceId,
        notificationDestinations.id,
        notificationDestinations.channel,
      ],
    }),
  }),
);

export const notificationPreferencesRelations = relations(notificationPreferences, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [notificationPreferences.workspaceId],
    references: [workspaces.id],
  }),
}));

export const notificationsRelations = relations(notifications, ({ one, many }) => ({
  workspace: one(workspaces, {
    fields: [notifications.workspaceId],
    references: [workspaces.id],
  }),
  deliveries: many(notificationDeliveries),
}));

export const notificationDeliveriesRelations = relations(notificationDeliveries, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [notificationDeliveries.workspaceId],
    references: [workspaces.id],
  }),
  notification: one(notifications, {
    fields: [notificationDeliveries.workspaceId, notificationDeliveries.notificationId],
    references: [notifications.workspaceId, notifications.id],
  }),
}));
