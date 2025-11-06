import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { ratelimit, requireWorkspace, t, withRatelimit } from "@/lib/trpc/trpc";
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

export const getBillingInfo = t.procedure
  .use(requireWorkspace)
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

    const [products, subscription, hasPreviousSubscriptions] = await Promise.all([
      stripe.products
        .list({
          active: true,
          ids: e.STRIPE_PRODUCT_IDS_PRO,
          limit: 100,
          expand: ["data.default_price"],
        })
        .then((res) => res.data.map(mapProduct).sort((a, b) => a.dollar - b.dollar)),
      ctx.workspace.stripeSubscriptionId
        ? await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId)
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

    return {
      products,
      subscription: subscription
        ? {
            id: subscription.id,
            status: subscription.status,
            cancelAt: subscription.cancel_at ? subscription.cancel_at * 1000 : undefined,
          }
        : undefined,
      hasPreviousSubscriptions,
      currentProductId: subscription?.items.data.at(0)?.plan.product?.toString() ?? undefined,
    };
  });
