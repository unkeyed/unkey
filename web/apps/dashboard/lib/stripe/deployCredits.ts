import type Stripe from "stripe";
import {
  type DeployBillingConfig,
  deployBillingConfig,
  deployBillingConfigured,
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

export { EXPIRY_GRACE_SECONDS };

/** Outcome of revoking the grants a reversed invoice funded. */
export type DeployCreditRevokeResult = {
  /** Grants voided, which Stripe only permits on a grant never applied to an invoice. */
  voided: number;
  /** Grants expired instead, because they had already been applied and could not be voided. */
  expired: number;
  /** Grants already void or already past their expiry, so there was no balance left to stop. */
  alreadyInactive: number;
};

/**
 * Voids the credit grants a given invoice funded, for when the payment behind it
 * is reversed: a refund, a dispute, a credit note, or the invoice being voided or
 * written off.
 *
 * Grants carry the funding invoice on `metadata.stripe_invoice_id`, and
 * creditGrants.list has no metadata filter, so this scans the customer's grants
 * the same way the duplicate check in [[grantDeployCreditsForInvoice]] does.
 * Voiding is idempotent: a grant that is already void or expired is counted and
 * skipped.
 *
 * Credit already consumed cannot be reclaimed. Voiding stops the remaining
 * balance being spent, which is the part still recoverable at this point.
 */
export async function revokeDeployCreditsForInvoice(
  stripe: Stripe,
  invoiceId: string,
): Promise<DeployCreditRevokeResult> {
  const invoice = await stripe.invoices.retrieve(invoiceId);
  const customerId =
    typeof invoice.customer === "string" ? invoice.customer : invoice.customer?.id;
  if (!customerId) {
    return { voided: 0, expired: 0, alreadyInactive: 0 };
  }

  return revokeGrants(stripe, customerId, (grant) => grant.metadata?.stripe_invoice_id === invoiceId);
}

/**
 * Voids every grant of a customer whose funding invoice no longer represents
 * money we kept.
 *
 * Used for charge and dispute reversals, which cannot name their invoice: this
 * Stripe API version has no `charge.invoice` and the payment intent does not
 * carry one either. Rather than guess, this re-reads each grant's funding invoice
 * and voids the grant when that invoice is no longer `paid`, or has been
 * refunded. Slightly more work than an exact match, and it self-heals: a grant
 * whose funding was reversed by any route gets caught the next time any reversal
 * event arrives for that customer.
 */
export async function revokeDeployCreditsForCustomer(
  stripe: Stripe,
  customerId: string,
): Promise<DeployCreditRevokeResult> {
  const settled = new Map<string, boolean>();

  const fundingReversed = async (invoiceId: string): Promise<boolean> => {
    const cached = settled.get(invoiceId);
    if (cached !== undefined) {
      return cached;
    }
    const invoice = await stripe.invoices.retrieve(invoiceId);
    // amount_remaining > 0 on a once-paid invoice means it was credited back.
    const reversed =
      invoice.status !== "paid" || (invoice.amount_remaining ?? 0) > 0;
    settled.set(invoiceId, reversed);
    return reversed;
  };

  return revokeGrants(stripe, customerId, async (grant) => {
    const invoiceId = grant.metadata?.stripe_invoice_id;
    if (!invoiceId) {
      return false;
    }
    return fundingReversed(invoiceId);
  });
}

/**
 * Shared scan: revoke every grant of a customer that `matches`.
 *
 * creditGrants.list has no metadata filter, so this walks the customer's grants
 * the same way the duplicate check in [[grantDeployCreditsForInvoice]] does.
 * Grants are roughly one per month, so this is a page or two.
 *
 * Void and expire are not interchangeable, and the difference decides this whole
 * feature. Stripe only allows voiding a grant "you haven't applied to an invoice,
 * either partially or completely"
 * (docs.stripe.com/billing/subscriptions/usage-based/billing-credits). The case
 * this code exists for is a customer who SPENT the credit and then reversed the
 * payment, so the grant is applied and void is rejected. Expiring is the
 * documented tool there: it kills whatever balance remains.
 *
 * So: try void, because it is the stronger statement and leaves a clearer audit
 * trail on an untouched grant, and fall back to expire when Stripe refuses.
 * Neither reclaims credit the customer already consumed; the only documented
 * route for that is voiding the invoice the credit was applied to, which is a
 * decision for a human, not this handler.
 */
async function revokeGrants(
  stripe: Stripe,
  customerId: string,
  matches: (grant: Stripe.Billing.CreditGrant) => boolean | Promise<boolean>,
): Promise<DeployCreditRevokeResult> {
  const nowSeconds = Math.floor(Date.now() / 1000);
  let voided = 0;
  let expired = 0;
  let alreadyInactive = 0;

  for await (const grant of stripe.billing.creditGrants.list({
    customer: customerId,
    limit: 100,
  })) {
    if (!(await matches(grant))) {
      continue;
    }
    // Already void, or already expired, means there is no balance left to stop.
    if (grant.voided_at || (grant.expires_at !== null && grant.expires_at <= nowSeconds)) {
      alreadyInactive++;
      continue;
    }

    try {
      await stripe.billing.creditGrants.voidGrant(grant.id);
      voided++;
    } catch (err) {
      if (!isAlreadyAppliedRejection(err)) {
        throw err;
      }
      // Applied to an invoice, so it cannot be voided. Expire it instead, which
      // stops the remaining balance being spent on anything further.
      await stripe.billing.creditGrants.expire(grant.id);
      expired++;
    }
  }

  return { voided, expired, alreadyInactive };
}

/**
 * Whether Stripe refused a void because the grant has already been applied to an
 * invoice. Matched loosely on the request-error class rather than a specific
 * message, because Stripe documents the restriction in prose but does not
 * document a stable error code for it. A genuine failure (auth, network, a
 * missing grant) is not a request-validation error and still propagates.
 */
function isAlreadyAppliedRejection(err: unknown): boolean {
  if (typeof err !== "object" || err === null) {
    return false;
  }
  const { type, rawType, statusCode } = err as {
    type?: unknown;
    rawType?: unknown;
    statusCode?: unknown;
  };
  const invalidRequest = type === "StripeInvalidRequestError" || rawType === "invalid_request_error";
  return invalidRequest && statusCode === 400;
}

/**
 * Whether an error is Stripe rejecting a request because the same idempotency
 * key is already in flight.
 *
 * Duck-typed rather than an `instanceof` check so this module keeps its
 * type-only Stripe import. Both shapes are accepted: stripe-node sets
 * `type` to "StripeIdempotencyError" and mirrors Stripe's own
 * "idempotency_error" on `rawType`.
 */
function isIdempotencyConflict(err: unknown): boolean {
  if (typeof err !== "object" || err === null) {
    return false;
  }
  const { type, rawType } = err as { type?: unknown; rawType?: unknown };
  return type === "StripeIdempotencyError" || rawType === "idempotency_error";
}

export type DeployCreditGrantResult =
  | {
      granted: false;
      reason: string;
      /**
       * True when the reason is transient and the caller should fail the webhook
       * so Stripe redelivers. Every other not-granted reason is terminal: acking
       * them is correct, because a retry produces the same answer.
       *
       * This distinction is the whole point. A grant that is skipped and acked is
       * gone forever, and the customer then pays full metered price on top of a
       * plan fee that was meant to cover it.
       */
      retryable?: boolean;
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

/** The invoice's Deploy plan-fee lines, matched by price id against the config. */
export function deployPlanFeeLines(
  config: DeployBillingConfig,
  lines: Stripe.InvoiceLineItem[],
): Stripe.InvoiceLineItem[] {
  return lines.filter((line) => {
    const priceId = line.pricing?.price_details?.price;
    return typeof priceId === "string" && Boolean(planForPlanFeePriceId(config, priceId));
  });
}

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
  const feeLines = deployPlanFeeLines(config, lines);
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
    // Null means two very different things and they must not be treated alike.
    //
    // Deploy is not configured here at all. The lookup-key env vars are optional
    // by design, so entire environments run without them, and every paid invoice
    // for every customer, API-only ones included, reaches this line in those.
    // Failing them would 500 on each one and, sustained, cost us the webhook
    // endpoint outright, taking down subscription handling that has nothing to do
    // with Deploy. Ack and move on.
    //
    // Or: the keys ARE set and one of them did not resolve to an active price.
    // That is a real misconfiguration, and acking it destroys the customer's
    // credit permanently with no way to reissue it. Since every Deploy
    // subscription renews on day 1, a bad catalogue would hit the whole fleet's
    // renewals at once. Retrying does not repair the catalogue by itself, but it
    // buys Stripe's multi-day retry window for someone else to, and it puts the
    // failure in the error rate rather than an info log nobody reads.
    if (deployBillingConfigured()) {
      return {
        granted: false,
        reason: "deploy billing configured but a price failed to resolve",
        retryable: true,
      };
    }
    return { granted: false, reason: "deploy billing not configured" };
  }
  if (!invoice.customer) {
    return { granted: false, reason: "invoice has no customer" };
  }
  const customerId = typeof invoice.customer === "string" ? invoice.customer : invoice.customer.id;

  // The payload carries only the first page of lines; a fee line split
  // onto page 2 would under-count the grant, so fetch every line.
  let lines = invoice.lines.data;
  if (invoice.lines.has_more) {
    console.warn("Invoice has more lines than the webhook payload carries; paginating", {
      invoiceId: invoice.id,
    });
    const allLines: Stripe.InvoiceLineItem[] = [];
    for await (const line of stripe.invoices.listLineItems(invoice.id, { limit: 100 })) {
      allLines.push(line);
    }
    lines = allLines;
  }

  const fee = netDeployFee(config, lines);
  if (!fee) {
    return { granted: false, reason: "no deploy plan-fee lines" };
  }
  if (fee.amountCents <= 0) {
    return { granted: false, reason: `non-positive net plan-fee amount (${fee.amountCents})` };
  }

  const expiresAt = fee.periodEnd + EXPIRY_GRACE_SECONDS;
  if (expiresAt * 1000 <= Date.now()) {
    // Paid long after the period closed; the usage invoice has already
    // finalized, so a grant could never be redeemed. Terminal, and there is no
    // repair path: the customer has paid a plan fee and will be billed the full
    // metered price the fee was meant to cover. Logged at error rather than info
    // because nothing else will ever surface it.
    console.error("Deploy credit grant skipped: period already closed", {
      invoiceId: invoice.id,
      customerId,
      feePeriodEnd: fee.periodEnd,
      expiresAt,
    });
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

  let created: Stripe.Billing.CreditGrant;
  try {
    created = await stripe.billing.creditGrants.create(
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
  } catch (err) {
    // Stripe rejects a request whose idempotency key is still in flight with a
    // 409 idempotency_error. Both invoice.paid and invoice.payment_succeeded fire
    // for a card-paid invoice and share this key, so concurrent delivery hits it
    // routinely. The other delivery is creating exactly this grant, so treat it
    // as done rather than turning it into a 500 that fails the webhook and
    // pollutes the error rate.
    if (isIdempotencyConflict(err)) {
      return {
        granted: false,
        reason: "grant creation already in flight for this invoice",
        periodTotalCents,
      };
    }
    throw err;
  }

  return {
    granted: true,
    grantId: created.id,
    amountCents: fee.amountCents,
    periodTotalCents: periodTotalCents + fee.amountCents,
  };
}

/**
 * The Deploy usage credit currently included for a customer, in cents: the sum
 * of their active metered credit grants for the current plan period.
 *
 * This is the allowance the estimated bill nets usage against. Reading it from
 * Stripe (rather than assuming it equals the catalog plan fee) is what makes
 * the card's estimate match the invoice: a mid-cycle plan change grants only
 * the prorated net fee ([[grantDeployCreditsForInvoice]]), so the catalog fee
 * would overstate the credit and under-report the bill.
 *
 * A period's grants (baseline plus any upgrade top-ups) share one expires_at,
 * and the current period's is the latest among unexpired grants. Summing only
 * that group avoids double-counting a previous period's grant during the short
 * post-period grace window, when both are briefly unexpired. Grants are roughly
 * one per month, so this is a page or two even for long-tenured customers.
 * Returns 0 when there is no active credit.
 */
export async function deployIncludedCreditCents(
  stripe: Stripe,
  customerId: string,
): Promise<number> {
  const nowSeconds = Date.now() / 1000;
  const centsByExpiry = new Map<number, number>();
  for await (const grant of stripe.billing.creditGrants.list({
    customer: customerId,
    limit: 100,
  })) {
    if (grant.applicability_config?.scope?.price_type !== "metered") {
      continue;
    }
    const expiresAt = grant.expires_at;
    if (expiresAt === null || expiresAt <= nowSeconds) {
      continue;
    }
    const value = grant.amount.monetary?.value ?? 0;
    centsByExpiry.set(expiresAt, (centsByExpiry.get(expiresAt) ?? 0) + value);
  }
  if (centsByExpiry.size === 0) {
    return 0;
  }
  const currentPeriodExpiry = Math.max(...centsByExpiry.keys());
  return centsByExpiry.get(currentPeriodExpiry) ?? 0;
}
