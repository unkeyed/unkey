import { auth } from "@/lib/auth/server";
import { getStripeClient } from "@/lib/stripe";
import { syncSubscriptionFromStripe } from "@/lib/stripe/sync";
import { TRPCError } from "@trpc/server";
import { workspaceProcedure } from "../../trpc";

export const cancelSubscription = workspaceProcedure.mutation(async ({ ctx }) => {
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
    cancel_at_period_end: true,
  });

  await syncSubscriptionFromStripe(stripe, ctx.workspace.id);
});
