import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import Stripe from "stripe";
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

const subscriptionSchema = z
  .object({
    id: z.string(),
    status: z.string(),
    cancelAt: z.number().optional(),
  })
  .optional();

const billingInfoSchema = z.object({
  products: z.array(productSchema),
  subscription: subscriptionSchema,
  hasPreviousSubscriptions: z.boolean(),
  currentProductId: z.string().optional(),
});

export const getBillingInfo = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(billingInfoSchema)
  .query(async ({ ctx }) => {
    const stripe = getStripeClient();
    const e = stripeEnv();
    if (!e) {
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Stripe is not configured",
      });
    }

    const [subscription, hasPreviousSubscriptions] = await Promise.all([
      ctx.workspace.stripeSubscriptionId
        ? // A stale recorded subscription id (cancelled and pruned from Stripe,
          // A stale id (lost deleted-webhook) 404s; treat as no subscription
          // instead of breaking the billing page. Anything else propagates.
          stripe.subscriptions
            .retrieve(ctx.workspace.stripeSubscriptionId)
            .catch((err: unknown) => {
              if (err instanceof Stripe.errors.StripeError && err.code === "resource_missing") {
                return undefined;
              }
              throw err;
            })
        : undefined,

      ctx.workspace.stripeCustomerId
        ? await stripe.subscriptions
            .list({
              customer: ctx.workspace.stripeCustomerId,
              status: "canceled",
            })
            .then((res) => res.data.length > 0)
        : false,
    ]);

    // The API subscription carries only the API plan item now, so items[0] is
    // it. Product via the item's price; the plan field is legacy.
    const apiItem = subscription ? subscription.items.data[0] : undefined;
    const apiProduct = apiItem?.price.product;
    const currentProductId = typeof apiProduct === "string" ? apiProduct : apiProduct?.id;

    // Check if user has an active enterprise subscription
    let enterpriseProductId: string | undefined;
    if (currentProductId && e.STRIPE_PRODUCT_IDS_ENTERPRISE.includes(currentProductId)) {
      enterpriseProductId = currentProductId;
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

    return {
      products,
      subscription: subscription
        ? {
            id: subscription.id,
            status: subscription.status,
            // Native cancel_at, set by cancelSubscription's cancel_at_period_end.
            cancelAt: subscription.cancel_at ? subscription.cancel_at * 1000 : undefined,
          }
        : undefined,
      hasPreviousSubscriptions,
      currentProductId,
    };
  });
