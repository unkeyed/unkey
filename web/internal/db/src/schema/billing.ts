import { relations } from "drizzle-orm";
import {
  bigint,
  boolean,
  int,
  json,
  mysqlEnum,
  mysqlTable,
  text,
  uniqueIndex,
  varchar,
} from "drizzle-orm/mysql-core";
import { identities } from "./identity";
import { lifecycleDates } from "./util/lifecycle_dates";
import { workspaces } from "./workspaces";

/**
 * One tiered-price step, matching the billingTier shape in @unkey/billing:
 * units firstUnit..lastUnit cost centsPerUnit each. lastUnit null means
 * unbounded; centsPerUnit null means free.
 */
export type RateCardTier = {
  firstUnit: number;
  lastUnit: number | null;
  centsPerUnit: string | null;
};

/**
 * Tiered prices per metered dimension. Omitted dimensions are not billed.
 */
export type RateCardConfig = {
  verifications?: RateCardTier[];
  credits?: RateCardTier[];
  ratelimits?: RateCardTier[];
};

/**
 * A customer-defined rate card: tiered prices over the metered dimensions
 * (VALID verifications, credits spent, passed ratelimit checks). Rate cards
 * make Unkey the pricing source of truth — the export API and the Stripe
 * Connect push both compute amounts from the same resolved card.
 */
export const rateCards = mysqlTable(
  "rate_cards",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    name: varchar("name", { length: 256 }).notNull(),
    /** ISO 4217, e.g. "USD". A rate card prices in exactly one currency. */
    currency: varchar("currency", { length: 3 }).notNull().default("USD"),
    config: json("config").$type<RateCardConfig>().notNull(),
    /**
     * Whether end-users may pick this card themselves (the allowed set).
     * Non-selectable cards can still be assigned by the workspace owner.
     */
    selectable: boolean("selectable").notNull().default(false),
    /**
     * Archived cards can no longer be assigned or selected, but remain
     * resolvable for periods they were recorded against.
     */
    archived: boolean("archived").notNull().default(false),
    ...lifecycleDates,
  },
  (table) => ({
    uniqueNamePerWorkspace: uniqueIndex("workspace_id_name_idx").on(table.workspaceId, table.name),
  }),
);

export const rateCardsRelations = relations(rateCards, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [rateCards.workspaceId],
    references: [workspaces.id],
  }),
}));

/**
 * Workspace-level end-user billing settings: the default rate card and the
 * Vault-encrypted Stripe Connect account reference. Credentials are stored
 * as ciphertext + encryption key id only — never plaintext. Unlinking a
 * connected account nulls both Stripe columns.
 */
export const workspaceBillingSettings = mysqlTable("workspace_billing_settings", {
  pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
  id: varchar("id", { length: 256 }).notNull().unique(),
  workspaceId: varchar("workspace_id", { length: 256 }).notNull().unique(),
  /**
   * Rate card in force when an identity has no assignment or selection.
   */
  defaultRateCardId: varchar("default_rate_card_id", { length: 256 }),
  /**
   * Vault-encrypted Stripe Connect account reference (acct_...), null until
   * onboarding starts.
   */
  stripeConnectEncrypted: text("stripe_connect_encrypted"),
  stripeConnectEncryptionKeyId: varchar("stripe_connect_encryption_key_id", { length: 256 }),
  /**
   * Onboarding state: "pending" from the moment a hosted onboarding session
   * starts, "linked" once Stripe reports details_submitted (or a verified
   * manual link completes). Only linked workspaces are billed at period
   * close. Null when no account reference is stored.
   */
  stripeConnectStatus: mysqlEnum("stripe_connect_status", ["pending", "linked"]),
  ...lifecycleDates,
});

export const workspaceBillingSettingsRelations = relations(workspaceBillingSettings, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [workspaceBillingSettings.workspaceId],
    references: [workspaces.id],
  }),
}));

/**
 * The rate card resolved for one identity and billing period, recorded the
 * first time the period is billed. Makes amounts traceable (an invoice always
 * names the exact card in force) and pins mid-period card changes to the next
 * period instead of silently re-pricing a closed one.
 */
export const billingPeriodRateCards = mysqlTable(
  "billing_period_rate_cards",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    identityId: varchar("identity_id", { length: 256 }).notNull(),
    year: int("year").notNull(),
    month: int("month").notNull(),
    rateCardId: varchar("rate_card_id", { length: 256 }).notNull(),
    /** Which precedence step produced the card (KTD7). */
    resolvedFrom: mysqlEnum("resolved_from", [
      "selection",
      "assignment",
      "workspace_default",
    ]).notNull(),
    /**
     * Unix millis when this identity+period was successfully pushed to the
     * billing provider. Set once, first-write-wins, after a successful push;
     * a set value makes the hourly period-close skip the identity so a closed
     * period is billed exactly once (defeats double-billing once Stripe's 24h
     * invoice-item idempotency window expires). Null until first pushed.
     */
    pushedAt: bigint("pushed_at", { mode: "number" }),
    ...lifecycleDates,
  },
  (table) => ({
    uniquePeriodPerIdentity: uniqueIndex("workspace_identity_period_idx").on(
      table.workspaceId,
      table.identityId,
      table.year,
      table.month,
    ),
  }),
);

export const billingPeriodRateCardsRelations = relations(billingPeriodRateCards, ({ one }) => ({
  workspace: one(workspaces, {
    fields: [billingPeriodRateCards.workspaceId],
    references: [workspaces.id],
  }),
  identity: one(identities, {
    fields: [billingPeriodRateCards.identityId],
    references: [identities.id],
  }),
  rateCard: one(rateCards, {
    fields: [billingPeriodRateCards.rateCardId],
    references: [rateCards.id],
  }),
}));

/**
 * Which of a workspace's keyspaces and ratelimit namespaces are enabled for
 * end-user billing. Enablement is opt-in: a resource is billed only when a row
 * exists here (presence == enabled). Usage on resources with no row is excluded
 * from an identity's billable quantities. `resource_id` is a keyspace id
 * (key_auth.id, the ClickHouse key_space_id) or a ratelimit namespace id
 * (ratelimit_namespaces.id, the ClickHouse namespace_id) per `resource_type`.
 */
export const billingBillableResources = mysqlTable(
  "billing_billable_resources",
  {
    pk: bigint("pk", { mode: "number", unsigned: true }).autoincrement().primaryKey(),
    id: varchar("id", { length: 256 }).notNull().unique(),
    workspaceId: varchar("workspace_id", { length: 256 }).notNull(),
    /** Whether resource_id names a keyspace (key_auth) or a ratelimit namespace. */
    resourceType: mysqlEnum("resource_type", ["keyspace", "namespace"]).notNull(),
    resourceId: varchar("resource_id", { length: 256 }).notNull(),
    ...lifecycleDates,
  },
  (table) => ({
    uniqueResourcePerWorkspace: uniqueIndex("workspace_resource_idx").on(
      table.workspaceId,
      table.resourceType,
      table.resourceId,
    ),
  }),
);

export const billingBillableResourcesRelations = relations(
  billingBillableResources,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [billingBillableResources.workspaceId],
      references: [workspaces.id],
    }),
  }),
);
