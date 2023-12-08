import type { Workspace } from "@unkey/db";
import { stripeEnv } from "../env";
export type Plan = NonNullable<Workspace["plan"]>;
export type Subscriptions = Workspace["subscriptions"];

export function defaultProSubscriptions(): Subscriptions {
  const env = stripeEnv();
  if (!env) {
    return null;
  }
  return {
    plan: {
      productId: env.STRIPE_PRODUCT_ID_PRO_PLAN,
      price: 25,
    },
    activeKeys: {
      productId: env.STRIPE_PRODUCT_ID_ACTIVE_KEYS,
      tiers: [
        {
          firstUnit: 1,
          lastUnit: 250,
          perUnit: 0,
        },
        {
          firstUnit: 251,
          lastUnit: null,
          perUnit: 0.1, // $0.10 per active key
        },
      ],
    },
    verifications: {
      productId: env.STRIPE_PRODUCT_ID_KEY_VERIFICATIONS,
      tiers: [
        {
          firstUnit: 1,
          lastUnit: 2500,
          perUnit: 0,
        },
        {
          firstUnit: 2501,
          lastUnit: null,
          perUnit: 0.0002, // $1 per 5k verifications
        },
      ],
    },
  };
}
