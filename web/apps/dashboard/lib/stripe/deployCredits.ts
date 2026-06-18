import type Stripe from "stripe";
import {
  type DeployBillingConfig,
  deployBillingConfig,
  planForPlanFeePriceId,
} from "./deployBilling";
import type { DeployPlan } from "./deployPlan";

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
  | {
      granted: false;
      reason: string;
      /**
       * The period's total granted credit (cents), recomputed from Stripe's
       * grants, when the invoice carries Deploy fee lines for an open period.
       * Present on the already-granted path so a redelivered webhook can
       * re-persist a total an earlier delivery failed to write. Undefined when
       * there is nothing to persist (no fee lines, closed period, no config).
       */
      periodTotalCents?: number;
    }
  | { granted: true; grantId: string; amountCents: number; periodTotalCents: number };

export type NetDeployFee = {
  /** Net of the invoice's Deploy plan-fee lines (cents); can be negative. */
  amountCents: number;
  /** Latest period end across the fee lines (unix seconds). */
  periodEnd: number;
  /** The plan the charged (largest) fee line maps to, when recognizable. */
  plan?: DeployPlan;
};

/**
 * Sums an invoice's Deploy plan-fee lines net of discounts, or returns null
 * when it has none.
 *
 * Summing the lines (rather than reading the catalog price) makes prorations
 * self-correcting across every flow: a mid-cycle subscribe grants the
 * prorated amount, a mid-cycle upgrade's always_invoice proration invoice
 * nets (+new fee, -unused old fee) to exactly the top-up, a downgrade nets
 * negative (no grant, no clawback), and a renewal grants the full fee.
 *
 * line.amount is the gross fee, excluding tax and discounts, so each line's
 * discount_amounts are subtracted (Stripe distributes invoice-level coupons
 * into these too) to track the fee actually paid, not the list price.
 */
export function netDeployFee(
  config: DeployBillingConfig,
  lines: Stripe.InvoiceLineItem[],
): NetDeployFee | null {
  const feeLines = lines.filter((line) => {
    const priceId = line.pricing?.price_details?.price;
    return typeof priceId === "string" && Boolean(planForPlanFeePriceId(config, priceId));
  });
  if (feeLines.length === 0) {
    return null;
  }

  // Label from the largest fee line. On an upgrade proration the lines are
  // +new prorated fee and -unused old fee in Stripe's own order, and the plan
  // being charged is the positive, larger one; picking feeLines[0] could name
  // the credit after the departing plan. Subscribe and renewal have a single
  // fee line, so this is a no-op there.
  const chargeLine = feeLines.reduce((max, line) => (line.amount > max.amount ? line : max));
  const chargePriceId = chargeLine.pricing?.price_details?.price;

  return {
    amountCents: feeLines.reduce((sum, line) => {
      const discounts = (line.discount_amounts ?? []).reduce((d, da) => d + da.amount, 0);
      return sum + line.amount - discounts;
    }, 0),
    periodEnd: Math.max(...feeLines.map((line) => line.period.end)),
    plan:
      typeof chargePriceId === "string" ? planForPlanFeePriceId(config, chargePriceId) : undefined,
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
  const config = await deployBillingConfig();
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

  // One pass over the customer's grants serves two purposes. Replay guard
  // beyond the 24h idempotency window: skip if this invoice already produced
  // a grant (creditGrants.list has no metadata filter, so match on the
  // invoice id). Period total: sum the grants that expire at this period's
  // boundary — a period's grants (subscribe or renewal baseline plus upgrade
  // top-ups) all share expires_at, and no other period's can, so the sum is
  // the period's true included credit regardless of how many deliveries or
  // replays got here first. Grants are roughly one per month, so this is a
  // page or two even for long-tenured customers.
  let duplicate: Stripe.Billing.CreditGrant | undefined;
  let periodTotalCents = 0;
  for await (const grant of stripe.billing.creditGrants.list({
    customer: customerId,
    limit: 100,
  })) {
    if (grant.expires_at === expiresAt) {
      periodTotalCents += grant.amount.monetary?.value ?? 0;
    }
    if (grant.metadata?.stripe_invoice_id === invoice.id) {
      duplicate = grant;
    }
  }
  if (duplicate) {
    return { granted: false, reason: `already granted (${duplicate.id})`, periodTotalCents };
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

  return {
    granted: true,
    grantId: created.id,
    amountCents: fee.amountCents,
    periodTotalCents: periodTotalCents + fee.amountCents,
  };
}
