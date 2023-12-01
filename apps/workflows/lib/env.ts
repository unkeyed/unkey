import { z } from "zod";

export const env = () =>
  z
    .object({
      DATABASE_HOST: z.string(),
      DATABASE_USERNAME: z.string(),
      DATABASE_PASSWORD: z.string(),
      TINYBIRD_TOKEN: z.string(),
      // HEARTBEAT_UPDATE_USAGE_URL: z.string().optional(),
      STRIPE_SECRET_KEY: z.string(),
      STRIPE_PRO_PLAN_PRICE_ID: z.string(),
      STRIPE_ACTIVE_KEYS_PRODUCT_ID: z.string(),
      STRIPE_ACTIVE_KEYS_PRICE_ID: z.string(),
      STRIPE_KEY_VERIFICATIONS_PRODUCT_ID: z.string(),
      STRIPE_KEY_VERIFICATIONS_PRICE_ID: z.string(),
    })
    .parse(process.env);
