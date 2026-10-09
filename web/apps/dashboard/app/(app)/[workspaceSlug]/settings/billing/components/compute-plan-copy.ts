import { formatDollars } from "@/lib/fmt";
import { COMPUTE_SIGNUP_CREDIT_CENTS } from "@/lib/stripe/computeSignupCreditAmount";
import type { DeployPlan } from "@/lib/stripe/deployPlan";
import { COMPUTE_BILLING_DOCS } from "@/lib/support";

/**
 * Plan fee minus the signup credit, never below zero. Stripe prorates the
 * first invoice, so the amount due today can be lower than this.
 */
export function firstInvoiceCents(planCents: number): number {
  return Math.max(0, planCents - COMPUTE_SIGNUP_CREDIT_CENTS);
}

export const FIRST_INVOICE_CREDIT_NOTE = `${formatDollars(COMPUTE_SIGNUP_CREDIT_CENTS)} on Unkey`;

/** Marketing copy for the Compute plan picker. */
export const PLAN_BLURBS: Record<DeployPlan, string> = {
  starter: "For hobby projects and testing ideas",
  pro: "For growing apps in production",
  business: "For teams scaling with confidence",
};

const MONTHLY_USAGE_CREDIT = "Every plan includes monthly usage credit.";

export const CREDITS_INFO = MONTHLY_USAGE_CREDIT;
export const CREDITS_LINK_LABEL = "See how credits work";
export const CREDITS_LINK_HREF = `${COMPUTE_BILLING_DOCS}#how-the-bill-works`;

const NOT_SUBSCRIBED_SUBTITLE =
  "Run and scale your projects. Every plan includes usage credits equal to its fee.";

export function planCreditsCopy(): string {
  return CREDITS_INFO;
}

export function deployNotSubscribedSubtitle(): string {
  return NOT_SUBSCRIBED_SUBTITLE;
}
