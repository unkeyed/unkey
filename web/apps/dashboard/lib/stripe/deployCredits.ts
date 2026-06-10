import type Stripe from "stripe";
import { deployBillingConfig, planForPlanFeePriceId } from "./deployPlans";
import { invoiceLinePriceId } from "./invoiceCompat";

/**
 * How long credits stay redeemable past the plan-fee period they belong to.
 *
 * A period's metered usage is billed on the invoice that finalizes at the
 * START of the next period (e.g. June usage bills on the July 1 invoice).
 * Credits that expired exactly at period end would therefore never apply to
 * the one invoice they exist for. Three days covers webhook-driven
 * finalization and the backup cron, while still expiring before the next
 * month's usage invoice, keeping the use-it-or-lose-it semantics.
 */
const EXPIRY_GRACE_SECONDS = 3 * 24 * 60 * 60;

export type DeployCreditGrantResult =
  | { granted: false; reason: string }
  | { granted: true; grantId: string; amountCents: number };

/**
 * Grants the Deploy usage credits a paid invoice entitles the customer to:
 * the net amount of its Deploy plan-fee lines, scoped to metered prices (the
 * only metered prices on a subscription are the Deploy meters; API tiers are
 * licensed), expiring shortly after the period the fee covers.
 *
 * Summing the plan-fee lines (rather than reading the catalog price) makes
 * prorations self-correcting: a mid-cycle subscribe grants the prorated
 * amount, and a renewal invoice carrying up/downgrade prorations grants the
 * net fee actually paid for the period. A net of zero or less grants nothing.
 *
 * Idempotent per invoice twice over: an idempotency key derived from the
 * invoice id covers webhook retries, and a metadata check against existing
 * grants covers replays beyond Stripe's 24h idempotency window.
 */
export async function grantDeployCreditsForInvoice(
  stripe: Stripe,
  invoice: Stripe.Invoice,
): Promise<DeployCreditGrantResult> {
  const config = deployBillingConfig();
  if (!config) {
    return { granted: false, reason: "deploy billing not configured" };
  }
  if (!invoice.customer) {
    return { granted: false, reason: "invoice has no customer" };
  }
  const customerId = typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id;

  // The webhook payload carries the first page of lines. Our subscriptions
  // have well under a page of items, so more pages means something unexpected;
  // log it rather than silently miscounting the fee.
  if (invoice.lines.has_more) {
    console.warn("Invoice has more lines than the webhook payload carries", {
      invoiceId: invoice.id,
    });
  }

  // Dual-shape price read: the invoice arrives via webhook, whose payload
  // shape follows the endpoint's pinned API version, not the SDK's.
  const feeLines = invoice.lines.data.filter((line) => {
    const priceId = invoiceLinePriceId(line);
    return Boolean(priceId && planForPlanFeePriceId(config, priceId));
  });
  if (feeLines.length === 0) {
    return { granted: false, reason: "no deploy plan-fee lines" };
  }

  const amountCents = feeLines.reduce((sum, line) => sum + line.amount, 0);
  if (amountCents <= 0) {
    return { granted: false, reason: `non-positive net plan-fee amount (${amountCents})` };
  }

  const periodEnd = Math.max(...feeLines.map((line) => line.period.end));
  const expiresAt = periodEnd + EXPIRY_GRACE_SECONDS;
  if (expiresAt * 1000 <= Date.now()) {
    // Paid long after the period closed; the usage invoice has already
    // finalized, so a grant could never be redeemed.
    return { granted: false, reason: "period already closed" };
  }

  // Replay guard beyond the idempotency window: skip if this invoice already
  // produced a grant.
  const existing = await stripe.billing.creditGrants.list({ customer: customerId, limit: 100 });
  const duplicate = existing.data.find((g) => g.metadata?.stripe_invoice_id === invoice.id);
  if (duplicate) {
    return { granted: false, reason: `already granted (${duplicate.id})` };
  }

  const firstFeeLine = feeLines[0];
  const firstFeePriceId = firstFeeLine ? invoiceLinePriceId(firstFeeLine) : undefined;
  const plan = firstFeePriceId ? planForPlanFeePriceId(config, firstFeePriceId) : undefined;

  // The grant name shows up as the credit line on the invoice, so it should
  // explain itself there: "Business plan monthly included usage ($50.00 off)".
  const planLabel = plan ? plan.charAt(0).toUpperCase() + plan.slice(1) : "Compute";

  const created = await stripe.billing.creditGrants.create(
    {
      name: `${planLabel} plan monthly included usage`,
      customer: customerId,
      category: "promotional",
      amount: {
        type: "monetary",
        monetary: { currency: invoice.currency, value: amountCents },
      },
      applicability_config: { scope: { price_type: "metered" } },
      expires_at: expiresAt,
      metadata: {
        stripe_invoice_id: invoice.id,
        ...(plan ? { deploy_plan: plan } : {}),
      },
    },
    { idempotencyKey: `deploy-credit-grant:${invoice.id}` },
  );

  return { granted: true, grantId: created.id, amountCents };
}
