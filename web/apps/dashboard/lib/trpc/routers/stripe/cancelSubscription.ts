import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findDeployItems } from "@/lib/stripe/deployPlans";
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

    // Cancelling at period end ends the WHOLE subscription — on a mixed
    // subscription that would silently take Compute (and its deployments)
    // down with the API plan. Until per-item scheduled cancellation exists,
    // require Compute to be cancelled first.
    const config = deployBillingConfig();
    if (config) {
      const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
      if (findDeployItems(config, sub.items.data).length > 0) {
        throw new TRPCError({
          code: "PRECONDITION_FAILED",
          message:
            "This subscription also carries your Compute plan. Cancel Compute first, then cancel the API plan.",
        });
      }
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
