import { getStripeClient } from "@/lib/stripe";
import { DEPLOY_PLANS, type DeployPlan } from "@/lib/stripe/deployPlan";
import { deployBillingConfig } from "@/lib/stripe/deployPlans";
import type Stripe from "stripe";
import { workspaceProcedure } from "../../trpc";

export type DeployPlanOption = {
  plan: DeployPlan;
  name: string;
  description: string | null;
  /** Plan-fee amount in the smallest currency unit (cents), or null if unset. */
  amount: number | null;
  currency: string;
  /** Recurring interval (e.g. "month"), or null for non-recurring prices. */
  interval: string | null;
};

/**
 * Lists the available Unkey Deploy plans with their plan-fee pricing, for the
 * subscribe / change UI. Unlike getDeploySubscription (which reads the local
 * deploy_plan signal), this is a catalog read and hits Stripe, mirroring
 * getProducts for the API plans. Returns configured=false when Deploy billing
 * is not set up so the UI can hide the section gracefully.
 */
export const getDeployPlans = workspaceProcedure.query(async () => {
  const config = deployBillingConfig();
  if (!config) {
    return { configured: false as const, plans: [] as DeployPlanOption[] };
  }

  const stripe = getStripeClient();
  const plans = await Promise.all(
    DEPLOY_PLANS.map(async (plan): Promise<DeployPlanOption> => {
      const price = await stripe.prices.retrieve(config.planFeePriceIds[plan], {
        expand: ["product"],
      });
      const product = price.product;
      const resolvedProduct =
        typeof product !== "string" && !product.deleted ? (product as Stripe.Product) : null;

      return {
        plan,
        name: resolvedProduct?.name ?? plan,
        description: resolvedProduct?.description ?? null,
        amount: price.unit_amount ?? null,
        currency: price.currency,
        interval: price.recurring?.interval ?? null,
      };
    }),
  );

  return { configured: true as const, plans };
});
