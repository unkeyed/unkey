import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findDeployItems } from "@/lib/stripe/deployBilling";
import { createApiCancelSchedule, getApiCancelSchedule } from "@/lib/stripe/subscriptionUtils";
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

    // Fail closed when config can't be resolved: a null config means we can't
    // tell whether this subscription carries Compute, so proceeding would risk
    // cancelling Compute along with the API plan. A transient resolution
    // failure is better surfaced as a retryable error than as a silent Compute
    // teardown.
    const config = await deployBillingConfig();
    if (!config) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Billing is temporarily unavailable. Please try again in a moment.",
      });
    }
    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
    const deployItems = findDeployItems(config, sub.items.data);

    if (deployItems.length > 0 && deployItems.length === sub.items.data.length) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "This subscription only carries Compute; there is no API plan to cancel.",
      });
    }

    if (deployItems.length === 0) {
      /**
       * API-only subscription: Stripe deletes it at period end. The webhook
       * handler for `customer.subscription.deleted` reverts tier/quotas and
       * deactivates all non-creator memberships, so we don't need to block
       * cancellation on member count here.
       */
      await stripe.subscriptions.update(ctx.workspace.stripeSubscriptionId, {
        cancel_at_period_end: true,
      });
      return;
    }

    // Mixed subscription: cancel_at_period_end would end the WHOLE subscription
    // and silently take Compute (and its deployments) down with the API plan,
    // and Stripe has no per-item scheduled cancellation. A subscription
    // schedule stands in for it: the current items run to period end, then a
    // Compute-only phase takes over. end_behavior "release" with one iteration
    // of that phase detaches the schedule afterwards, returning the
    // subscription to normal item-level management. The subscription.updated
    // webhook downgrades tier/quotas when the phase boundary hits, and
    // uncancelSubscription resumes by releasing the schedule.
    if (sub.schedule) {
      const existing = await getApiCancelSchedule(stripe, sub);
      if (existing) {
        // Cancellation is already pending; nothing to do.
        return;
      }
      // Some other schedule manages this subscription — mutating its phases
      // could clobber whatever it encodes.
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "This subscription is managed by a billing schedule. Contact support@unkey.com to cancel.",
      });
    }

    const deployItemIds = new Set(deployItems.map((item) => item.id));
    await createApiCancelSchedule(
      stripe,
      sub.id,
      sub.items.data.filter((item) => deployItemIds.has(item.id)),
    );
  });
