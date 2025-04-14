import { auth } from "@/lib/auth/server";
import { stripeEnv } from "@/lib/env";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
import { requireUser, requireWorkspace, t } from "../../trpc";
export const cancelSubscription = t.procedure
  .use(requireUser)
  .use(requireWorkspace)
  .mutation(async ({ ctx }) => {
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({ code: "INTERNAL_SERVER_ERROR", message: "Stripe is not set up" });
    }

    const memberships = await auth.getOrganizationMemberList(ctx.workspace.orgId).catch((err) => {
      console.error(err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch organization members",
      });
    });
    if (memberships.data.length > 1) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message:
          "Workspace has more than one member. You must remove all other members before downgrading to the free tier.",
      });
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
