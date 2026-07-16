import { getStripeClient } from "@/lib/stripe";
import { getApiCancelSchedule } from "@/lib/stripe/subscriptionUtils";
import { TRPCError } from "@trpc/server";
import { requireWorkspaceAdmin, workspaceProcedure } from "../../trpc";

export const uncancelSubscription = workspaceProcedure
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

    const sub = await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId);

    // A mixed-subscription API cancel is a scheduled phase-out, not
    // cancel_at_period_end (see cancelSubscription). Resuming means releasing
    // the schedule: the subscription detaches and continues with its current
    // (API + Compute) items as if nothing had been scheduled.
    const apiCancelSchedule = await getApiCancelSchedule(stripe, sub);
    if (apiCancelSchedule) {
      if (apiCancelSchedule.status === "active" || apiCancelSchedule.status === "not_started") {
        await stripe.subscriptionSchedules.release(apiCancelSchedule.id);
      }
      return;
    }

    await stripe.subscriptions.update(ctx.workspace.stripeSubscriptionId, {
      cancel_at_period_end: false,
    });
  });
