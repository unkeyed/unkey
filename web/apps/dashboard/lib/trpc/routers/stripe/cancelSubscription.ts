import { TRPCError } from "@trpc/server";
import { getStripeClient } from "@/lib/stripe";
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

    const subscription = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);
    if (subscription.schedule) {
      const scheduleId =
        typeof subscription.schedule === "string"
          ? subscription.schedule
          : subscription.schedule.id;
      // Cancellation supersedes any pending plan change. Release the schedule
      // first so the native period-end cancellation remains authoritative.
      await stripe.subscriptionSchedules.release(scheduleId);
    }

    // The API product owns its own subscription, so cancelling is a native
    // whole-subscription cancel at period end. Stripe deletes it at the boundary
    // and the customer.subscription.deleted webhook reverts tier/quotas and
    // deactivates non-creator memberships, so no member-count block is needed here.
    await stripe.subscriptions.update(subscription.id, {
      cancel_at_period_end: true,
    });
  });
