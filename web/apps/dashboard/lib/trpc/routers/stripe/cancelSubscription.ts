import { getStripeClient } from "@/lib/stripe";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export const cancelSubscription = workspaceProcedure
  .use(requireWorkspaceAdmin)
  .mutation(async ({ ctx }) => {
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

    // The API product owns its own subscription, so cancelling is a native
    // whole-subscription cancel at period end. Stripe deletes it at the boundary
    // and the customer.subscription.deleted webhook reverts tier/quotas and
    // deactivates non-creator memberships, so no member-count block is needed here.
    await stripe.subscriptions.update(ctx.workspace.stripeSubscriptionId, {
      cancel_at_period_end: true,
    });
  });
