import { getStripeClient } from "@/lib/stripe";
import { syncSubscriptionFromStripe } from "@/lib/stripe/sync";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";

export const syncSubscription = workspaceProcedure.mutation(async ({ ctx }) => {
  if (!ctx.workspace.stripeSubscriptionId) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "Workspace doesn't have a subscription to sync",
    });
  }

  const stripe = getStripeClient();
  await syncSubscriptionFromStripe(stripe, ctx.workspace.id);
});
