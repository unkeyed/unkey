import { getStripeClient } from "@/lib/stripe";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";

export const cancelSubscription = workspaceProcedure.mutation(async ({ ctx }) => {
  const stripe = getStripeClient();

  if (!ctx.workspace.stripeCustomerId) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "Workspace doesn't have a stripe customer id.",
    });
  }
  if (!ctx.workspace.stripeSubscriptionId) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "Workspace doesn't have a stripe subscrption id.",
    });
  }

  /**
   * Stripe deletes the subscription at period end. The webhook handler for
   * `customer.subscription.deleted` reverts tier/quotas and deactivates all non-creator
   * memberships, so we don't need to block cancellation on member count here.
   */
  await stripe.subscriptions.update(ctx.workspace.stripeSubscriptionId, {
    cancel_at_period_end: true,
  });
});
