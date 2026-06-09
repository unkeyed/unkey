import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findApiItem } from "@/lib/stripe/deployPlans";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { mapProduct } from "../utils/stripe";

const productSchema = z.object({
  id: z.string(),
  name: z.string(),
  priceId: z.string(),
  dollar: z.number(),
  quotas: z.object({
    requestsPerMonth: z.number(),
  }),
});

export const getProducts = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(z.array(productSchema))
  .query(async ({ ctx }) => {
    const stripe = getStripeClient();
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe is not configured",
      });
    }

    // Check if user has an active enterprise subscription
    let enterpriseProductId: string | undefined;
    if (ctx.workspace.stripeSubscriptionId) {
      try {
        const subscription = await stripe.subscriptions.retrieve(
          ctx.workspace.stripeSubscriptionId,
        );
        // The API item, skipping Deploy items (items[0] is a Deploy price on
        // a Compute-first subscription); product via price, plan is legacy.
        const apiItem = findApiItem(deployBillingConfig(), subscription.items.data);
        const product = apiItem?.price.product;
        const currentProductId = typeof product === "string" ? product : product?.id;
        if (currentProductId && e.STRIPE_PRODUCT_IDS_ENTERPRISE.includes(currentProductId)) {
          enterpriseProductId = currentProductId;
        }
      } catch (error) {
        // If subscription retrieval fails, default to showing only Pro products
        console.error("Failed to retrieve subscription:", error);
      }
    }

    const productIds = enterpriseProductId
      ? [...e.STRIPE_PRODUCT_IDS_PRO, enterpriseProductId]
      : e.STRIPE_PRODUCT_IDS_PRO;

    const products = await stripe.products
      .list({
        active: true,
        ids: productIds,
        limit: 100,
        expand: ["data.default_price"],
      })
      .then((res) => res.data.map(mapProduct).sort((a, b) => a.dollar - b.dollar));

    return products;
  });
