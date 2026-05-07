import { getStripeClient } from "@/lib/stripe";
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

    await stripe.subscriptions.update(ctx.workspace.stripeSubscriptionId, {
      cancel_at_period_end: false,
    });
  });
