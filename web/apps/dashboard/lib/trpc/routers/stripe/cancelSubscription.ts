import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findDeployItems } from "@/lib/stripe/deployBilling";
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
    //
    // Fail closed when config can't be resolved: a null config means we can't
    // tell whether this subscription carries Compute, so proceeding would skip
    // the guard and risk cancelling Compute along with the API plan. A transient
    // resolution failure is better surfaced as a retryable error than as a
    // silent Compute teardown.
    const config = await deployBillingConfig();
    if (!config) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Billing is temporarily unavailable. Please try again in a moment.",
      });
    }
    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
    if (findDeployItems(config, sub.items.data).length > 0) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "This subscription also carries your Compute plan. Cancel Compute first, then cancel the API plan.",
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
