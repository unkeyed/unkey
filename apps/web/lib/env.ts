import { z } from "zod";

export const env = () =>
  z
    .object({
      VERCEL_ENV: z
        .enum(["development", "preview", "production"])
        .optional()
        .default("development"),
      VERCEL_URL: z.string().optional(),

      UNKEY_WORKSPACE_ID: z.string().min(1),
      UNKEY_API_ID: z.string().min(1),

      UPSTASH_REDIS_REST_URL: z.string().optional(),
      UPSTASH_REDIS_REST_TOKEN: z.string().optional(),

      TINYBIRD_TOKEN: z.string().optional(),

      UNKEY_API_URL: z.string().url().default("http://127.0.0.1:8080"),
      NEXT_PUBLIC_UNKEY_API_URL: z.string().url().default("http://127.0.0.1:8080"),
      UNKEY_APP_AUTH_TOKEN: z.string().min(1),

      CLERK_WEBHOOK_SECRET: z.string().optional(),
      RESEND_API_KEY: z.string().optional(),
      RESEND_AUDIENCE_ID: z.string().optional(),

      UPTIME_CRON_URL_COLLECT_BILLING: z.string().optional(),
      PLAIN_API_KEY: z.string().optional(),

      NEXT_PUBLIC_CLERK_PUBLISHABLE_KEY: z.string().min(1),
      CLERK_SECRET_KEY: z.string().min(1),
    })
    .parse(process.env);

export const dbEnv = () =>
  z
    .object({
      DATABASE_HOST: z.string().min(1),
      DATABASE_USERNAME: z.string().min(1),
      DATABASE_PASSWORD: z.string().min(1),
    })
    .parse(process.env);

export const vercelIntegrationSchema = z.object({
  VERCEL_INTEGRATION_CLIENT_ID: z.string(),
  VERCEL_INTEGRATION_CLIENT_SECRET: z.string(),
});

const vercelIntegrationParsed = vercelIntegrationSchema.safeParse(process.env);
export const vercelIntegrationEnv = () =>
  vercelIntegrationParsed.success ? vercelIntegrationParsed.data : null;

const stripeSchema = z.object({
  STRIPE_SECRET_KEY: z.string(),
  STRIPE_WEBHOOK_SECRET: z.string(),
  STRIPE_PRO_PLAN_PRICE_ID: z.string(),
  STRIPE_ACTIVE_KEYS_PRODUCT_ID: z.string(),
  STRIPE_ACTIVE_KEYS_PRICE_ID: z.string(),
  STRIPE_KEY_VERIFICATIONS_PRODUCT_ID: z.string(),
  STRIPE_KEY_VERIFICATIONS_PRICE_ID: z.string(),
});

const stripeParsed = stripeSchema.safeParse(process.env);
export const stripeEnv = () => (stripeParsed.success ? stripeParsed.data : null);
