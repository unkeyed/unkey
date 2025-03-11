import { stripeEnv } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { auth, t } from "../../trpc";
export const cancelSubscription = t.procedure.use(auth).mutation(async ({ ctx }) => {
  const e = stripeEnv();
  if (!e) {
    throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Stripe is not set up" });
  }

  const stripe = new Stripe(e.STRIPE_SECRET_KEY, {
    apiVersion: "2023-10-16",
    typescript: true,
  });

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
    cancel_at_period_end: true,
  });
});
