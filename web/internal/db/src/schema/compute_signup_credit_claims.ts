import { relations } from "drizzle-orm";
import { bigint, mysqlTable, uniqueIndex } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";
import { workspaces } from "./workspaces";

/**
 * One Compute signup credit per WorkOS user, per workspace, and per card fingerprint.
 *
 * Customer balance transactions can only be listed per customer, so a second
 * credit cannot be rejected from Stripe alone once the 24-hour idempotency key
 * is gone. This row is inserted before `customers.createBalanceTransaction`.
 * `attempt_id` is the compare-and-swap token: the balance transaction runs only
 * after an update matches the attempt that was read, and a delete matches only
 * that same token. Stripe remains the source of truth for the credit; the row
 * is only the lock.
 *
 * The user id is taken only from the SetupIntent for this card, or from a
 * subscription whose default payment method or latest invoice payment method
 * is this card. A portal or Dashboard card add without that metadata gets no
 * credit. An unfinished row for a different workspace, card, or user is left
 * in place and logged for manual reconcile.
 */
export const computeSignupCreditClaims = mysqlTable(
  "compute_signup_credit_claims",
  {
    pk: primaryKey(),
    cardFingerprint: caseSensitiveVarchar("card_fingerprint", {
      length: 128,
    }).notNull(),
    workspaceId: id("workspace_id").notNull(),
    stripeCustomerId: caseSensitiveVarchar("stripe_customer_id", { length: 256 }).notNull(),
    stripeBalanceTransactionId: caseSensitiveVarchar("stripe_balance_transaction_id", {
      length: 256,
    }),
    workosUserId: caseSensitiveVarchar("workos_user_id", {
      length: 256,
    }).notNull(),
    attemptId: caseSensitiveVarchar("attempt_id", { length: 64 }).notNull(),
    createdAt: bigint("created_at", { mode: "number" }).notNull(),
  },
  (table) => [
    uniqueIndex("compute_signup_credit_claims_fingerprint_uq").on(table.cardFingerprint),
    uniqueIndex("compute_signup_credit_claims_workspace_uq").on(table.workspaceId),
    uniqueIndex("compute_signup_credit_claims_user_uq").on(table.workosUserId),
  ],
);

export const computeSignupCreditClaimsRelations = relations(
  computeSignupCreditClaims,
  ({ one }) => ({
    workspace: one(workspaces, {
      fields: [computeSignupCreditClaims.workspaceId],
      references: [workspaces.id],
    }),
  }),
);
