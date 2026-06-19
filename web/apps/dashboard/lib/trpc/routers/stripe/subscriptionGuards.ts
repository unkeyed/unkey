import { TRPCError } from "@trpc/server";
import type Stripe from "stripe";

/**
 * Guards that a pre-existing subscription can safely have a new billed item
 * attached. Stripe silently accepts attaching an item to a subscription that is
 * scheduled to cancel (the item then never bills next cycle), and to a non-USD
 * subscription, so these are checked explicitly rather than left to
 * error_if_incomplete. Used by both subscribeDeploy (adding Deploy items) and
 * createSubscription (adding the API plan) so the two append paths stay in
 * lock-step. Decided in ENG-2876.
 */
export function assertSubscriptionAttachable(sub: Stripe.Subscription): void {
  if (sub.status !== "active" && sub.status !== "trialing") {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: `Your subscription is ${sub.status.replace(/_/g, " ")}. Settle any outstanding invoices before changing your plan.`,
    });
  }
  if (sub.cancel_at_period_end) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message:
        "Your subscription is set to cancel at the end of this period. Resume it before adding a plan, or wait until it ends to start fresh.",
    });
  }
  if (sub.currency !== "usd") {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message:
        "Prices are in USD and cannot be added to a non-USD subscription. Contact support@unkey.com.",
    });
  }
}
