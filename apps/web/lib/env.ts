import { z } from "zod";

export const envSchema = z.object({
  VERCEL_ENV: z.enum(["development", "preview", "production"]).optional().default("development"),
  VERCEL_URL: z.string().optional(),

  UNKEY_WORKSPACE_ID: z.string(),
  UNKEY_API_ID: z.string(),

  UPSTASH_REDIS_REST_URL: z.string().optional(),
  UPSTASH_REDIS_REST_TOKEN: z.string().optional(),

  TINYBIRD_TOKEN: z.string(),

  // Clerk Org ID
  TENANT_ID: z.string().min(1),

  UNKEY_API_URL: z.string().url().default("https://api.unkey.dev"),
  NEXT_PUBLIC_UNKEY_API_URL: z.string().url().default("https://api.unkey.dev"),
  UNKEY_APP_AUTH_TOKEN: z.string(),

  CLERK_WEBHOOK_SECRET: z.string().optional(),
  RESEND_API_KEY: z.string().optional(),
  RESEND_AUDIENCE_ID: z.string().optional(),

  UPTIME_CRON_URL_COLLECT_BILLING: z.string().optional(),
  PLAIN_API_KEY: z.string().optional(),
});

export const dbSchema = z.object({
  DATABASE_HOST: z.string().min(1),
  DATABASE_USERNAME: z.string().min(1),
  DATABASE_PASSWORD: z.string().min(1),
  DATABASE_NAME: z.string().min(1),
});

export const env = () => envSchema.parse(process.env);

export const dbEnv = () => dbSchema.parse(process.env);

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
