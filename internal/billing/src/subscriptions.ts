import { z } from "zod";
import { billingTier } from "./tiers";

const fixedSubscriptionSchema = z.object({
  productId: z.string(),
  cents: z.string().regex(/^\d{1,15}(\.\d{1,12})?$/), // in cents, e.g. "10.124" = $0.10124
});
export type FixedSubscription = z.infer<typeof fixedSubscriptionSchema>;

const tieredSubscriptionSchema = z.object({
  productId: z.string(),
  tiers: z.array(billingTier),
});

export type TieredSubscription = z.infer<typeof tieredSubscriptionSchema>;

export const subscriptionsSchema = z.object({
  activeKeys: tieredSubscriptionSchema.optional(),
  verifications: tieredSubscriptionSchema.optional(),
  plan: fixedSubscriptionSchema.optional(),
  support: fixedSubscriptionSchema.optional(),
});

export type Subscriptions = z.infer<typeof subscriptionsSchema>;

export function defaultProSubscriptions(): Subscriptions | null {
  const stripeEnv = z.object({
    STRIPE_PRODUCT_ID_PRO_PLAN: z.string(),
    STRIPE_PRODUCT_ID_ACTIVE_KEYS: z.string(),
    STRIPE_PRODUCT_ID_KEY_VERIFICATIONS: z.string(),
    STRIPE_PRODUCT_ID_SUPPORT: z.string(),
  });
  const env = stripeEnv.parse(process.env);
  if (!env) {
    return null;
  }
  return {
    plan: {
      productId: env.STRIPE_PRODUCT_ID_PRO_PLAN,
      cents: "2500", // $25
    },
    activeKeys: {
      productId: env.STRIPE_PRODUCT_ID_ACTIVE_KEYS,
      tiers: [
        {
          firstUnit: 1,
          lastUnit: 250,
          centsPerUnit: null,
        },
        {
          firstUnit: 251,
          lastUnit: null,
          centsPerUnit: "10", // $0.10 per active key
        },
      ],
    },
    verifications: {
      productId: env.STRIPE_PRODUCT_ID_KEY_VERIFICATIONS,
      tiers: [
        {
          firstUnit: 1,
          lastUnit: 100_000,
          centsPerUnit: null,
        },
        {
          firstUnit: 100_001,
          lastUnit: null,
          centsPerUnit: "0.01", // $0.0001 per verification or  $10 per 100k verifications
        },
      ],
    },
  };
}
