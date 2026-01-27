import { stripeEnv } from "@/lib/env";
import { getStripeClient } from "@/lib/stripe";
import { syncSubscriptionFromStripe } from "@/lib/stripe/sync";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";
import { mapProduct } from "../utils/stripe";

const GRACE_PERIOD_MS = 7 * 24 * 60 * 60 * 1000; // 7 days

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
  isInGracePeriod: z.boolean(),
  gracePeriodEndsAt: z.number().optional(),
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

    if (ctx.workspace.stripeSubscriptionId) {
      await syncSubscriptionFromStripe(stripe, ctx.workspace.id);
    }

    const [subscription, hasPreviousSubscriptions] = await Promise.all([
      ctx.workspace.stripeSubscriptionId
        ? await stripe.subscriptions.retrieve(ctx.workspace.stripeSubscriptionId).catch((error) => {
            if (error instanceof Error && "code" in error && error.code === "resource_missing") {
              console.warn(
                `Subscription ${ctx.workspace.stripeSubscriptionId} not found in Stripe for workspace ${ctx.workspace.id}`,
              );
              return undefined;
            }
            throw error;
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

    let enterpriseProductId: string | undefined;
    try {
      const currentProductId = subscription?.items.data.at(0)?.plan.product?.toString();
      if (currentProductId && e.STRIPE_PRODUCT_IDS_ENTERPRISE.includes(currentProductId)) {
        enterpriseProductId = currentProductId;
      }
    } catch (error) {
      console.error("Error checking enterprise subscription:", error);
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

    const now = Date.now();
    const isInGracePeriod =
      ctx.workspace.paymentFailedAt !== null &&
      ctx.workspace.paymentFailedAt !== undefined &&
      now - ctx.workspace.paymentFailedAt < GRACE_PERIOD_MS;

    const gracePeriodEndsAt = ctx.workspace.paymentFailedAt
      ? ctx.workspace.paymentFailedAt + GRACE_PERIOD_MS
      : undefined;

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
      isInGracePeriod,
      gracePeriodEndsAt,
    };
  });
