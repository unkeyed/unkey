import { z } from "zod";

export function env() {
  const parsed = z
    .object({
      DATABASE_HOST: z.string(),
      DATABASE_USERNAME: z.string(),
      DATABASE_PASSWORD: z.string(),
      TINYBIRD_TOKEN: z.string(),
      STRIPE_SECRET_KEY: z.string(),

      STRIPE_PRODUCT_ID_KEY_VERIFICATIONS: z.string(),
      STRIPE_PRODUCT_ID_ACTIVE_KEYS: z.string(),
      STRIPE_PRODUCT_ID_PRO_PLAN: z.string(),
      STRIPE_PRODUCT_ID_SUPPORT: z.string(),

      CLERK_SECRET_KEY: z.string(),

      RESEND_API_KEY: z.string(),
      TRIGGER_API_KEY: z.string(),
    })
    .safeParse(process.env);
  if (!parsed.success) {
    throw new Error(`env: ${parsed.error.message}`);
  }
  return parsed.data;
}
