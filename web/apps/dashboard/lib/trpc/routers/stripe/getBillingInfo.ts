import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { deployBillingConfig, findApiItem } from "@/lib/stripe/deployBilling";
import { getApiCancelSchedule } from "@/lib/stripe/subscriptionUtils";
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

    // The API plan item, skipping Deploy items: on a Compute-first
    // subscription items[0] is a Deploy price, not the API plan.
    const apiItem = subscription
      ? findApiItem(await deployBillingConfig(), subscription.items.data)
      : undefined;
    // Product via the item's price; the plan field is legacy.
    const apiProduct = apiItem?.price.product;
    const currentProductId = typeof apiProduct === "string" ? apiProduct : apiProduct?.id;

    // A mixed-subscription API cancel is a scheduled phase-out with no
    // cancel_at on the subscription (see cancelSubscription); surface its
    // phase boundary as cancelAt so the pending-cancellation banner and the
    // resume flow work unchanged. Only while the API item is still present —
    // once the boundary passes, the plan is simply gone.
    let scheduledApiCancelAt: number | undefined;
    if (subscription && apiItem) {
      const apiCancelSchedule = await getApiCancelSchedule(stripe, subscription);
      if (apiCancelSchedule?.current_phase?.end_date) {
        scheduledApiCancelAt = apiCancelSchedule.current_phase.end_date * 1000;
      }
    }

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
            cancelAt: subscription.cancel_at ? subscription.cancel_at * 1000 : scheduledApiCancelAt,
          }
        : undefined,
      hasPreviousSubscriptions,
      currentProductId,
    };
  });
