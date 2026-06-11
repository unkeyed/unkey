import type Stripe from "stripe";
import type { DeployPlan } from "./deployPlan";
import {
  type DeployBillingConfig,
  deployBillingConfig,
  planForPlanFeePriceId,
} from "./deployPlans";

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

export type NetDeployFee = {
  /** Net of the invoice's Deploy plan-fee lines (cents); can be negative. */
  amountCents: number;
  /** Latest period end across the fee lines (unix seconds). */
  periodEnd: number;
  /** The plan the first fee line maps to, when recognizable. */
  plan?: DeployPlan;
};

/**
 * Sums an invoice's Deploy plan-fee lines, or returns null when it has none.
 *
 * Summing the lines (rather than reading the catalog price) makes prorations
 * self-correcting across every flow: a mid-cycle subscribe grants the
 * prorated amount, a mid-cycle upgrade's always_invoice proration invoice
 * nets (+new fee, -unused old fee) to exactly the top-up, a downgrade nets
 * negative (no grant, no clawback), and a renewal grants the full fee.
 *
 */
export function netDeployFee(
  config: DeployBillingConfig,
  lines: Stripe.InvoiceLineItem[],
): NetDeployFee | null {
  const feeLines = lines.filter((line) => {
    const priceId = line.pricing?.price_details?.price;
    return typeof priceId === "string" && Boolean(planForPlanFeePriceId(config, priceId));
  });
  const firstFeeLine = feeLines[0];
  if (!firstFeeLine) {
    return null;
  }

  const firstFeePriceId = firstFeeLine.pricing?.price_details?.price;

  return {
    amountCents: feeLines.reduce((sum, line) => sum + line.amount, 0),
    periodEnd: Math.max(...feeLines.map((line) => line.period.end)),
    plan:
      typeof firstFeePriceId === "string"
        ? planForPlanFeePriceId(config, firstFeePriceId)
        : undefined,
  };
}

/**
 * Grants the Deploy usage credits a paid invoice entitles the customer to:
 * the net amount of its Deploy plan-fee lines ([[netDeployFee]]), scoped to
 * metered prices (the only metered prices on a subscription are the Deploy
 * meters; API tiers are licensed), expiring shortly after the period the fee
 * covers. A net of zero or less grants nothing.
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

  const fee = netDeployFee(config, invoice.lines.data);
  if (!fee) {
    return { granted: false, reason: "no deploy plan-fee lines" };
  }
  if (fee.amountCents <= 0) {
    return { granted: false, reason: `non-positive net plan-fee amount (${fee.amountCents})` };
  }

  const expiresAt = fee.periodEnd + EXPIRY_GRACE_SECONDS;
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

  // The grant name shows up as the credit line on the invoice, so it should
  // explain itself there: "Business plan monthly included usage ($50.00 off)".
  const planLabel = fee.plan ? fee.plan.charAt(0).toUpperCase() + fee.plan.slice(1) : "Compute";

  const created = await stripe.billing.creditGrants.create(
    {
      name: `${planLabel} plan monthly included usage`,
      customer: customerId,
      category: "promotional",
      amount: {
        type: "monetary",
        monetary: { currency: invoice.currency, value: fee.amountCents },
      },
      applicability_config: { scope: { price_type: "metered" } },
      expires_at: expiresAt,
      metadata: {
        stripe_invoice_id: invoice.id,
        ...(fee.plan ? { deploy_plan: fee.plan } : {}),
      },
    },
    { idempotencyKey: `deploy-credit-grant:${invoice.id}` },
  );

  return { granted: true, grantId: created.id, amountCents: fee.amountCents };
}
