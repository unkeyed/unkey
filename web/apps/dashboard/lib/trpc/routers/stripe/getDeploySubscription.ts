import { getStripeClient } from "@/lib/stripe";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import Stripe from "stripe";
import { workspaceProcedure } from "../../trpc";

/**
 * Returns the workspace's current Unkey Deploy plan plus recoverable payment
 * state. The plan remains a local entitlement signal; Stripe is read only when
 * a subscription is recorded so an incomplete first payment can be resumed.
 */
export const getDeploySubscription = workspaceProcedure.query(async ({ ctx }) => {
  const raw = ctx.workspace.deployPlan;
  const plan: DeployPlan | null =
    raw && (DEPLOY_PLANS as readonly string[]).includes(raw) ? (raw as DeployPlan) : null;

  if (!ctx.workspace.stripeDeploySubscriptionId) {
    return { plan, status: null };
  }

  const stripe = getStripeClient();
  const subscription = await stripe.subscriptions
    .retrieve(ctx.workspace.stripeDeploySubscriptionId)
    .catch((error: unknown) => {
      if (error instanceof Stripe.errors.StripeError && error.code === "resource_missing") {
        return null;
      }
      throw error;
    });
  if (!subscription) {
    return { plan, status: null };
  }

  return {
    plan,
    status: subscription.status,
  };
});
