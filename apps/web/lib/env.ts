import { z } from "zod";

export const env = z
  .object({
    VERCEL_ENV: z.enum(["development", "preview", "production"]).optional().default("development"),
    VERCEL_URL: z.string().optional(),
    DATABASE_HOST: z.string(),
    DATABASE_USERNAME: z.string(),
    DATABASE_PASSWORD: z.string(),

    UNKEY_WORKSPACE_ID: z.string(),
    UNKEY_API_ID: z.string(),

    UPSTASH_KAFKA_REST_URL: z.string(),
    UPSTASH_KAFKA_REST_USERNAME: z.string(),
    UPSTASH_KAFKA_REST_PASSWORD: z.string(),

    UPSTASH_REDIS_REST_URL: z.string().optional(),
    UPSTASH_REDIS_REST_TOKEN: z.string().optional(),

    TINYBIRD_TOKEN: z.string(),
   
    UNKEY_API_URL: z.string().url().default("https://api.unkey.dev"),
    UNKEY_APP_AUTH_TOKEN: z.string(),

    CLERK_WEBHOOK_SECRET: z.string().optional(),
    LOOPS_API_KEY: z.string().optional(),
  })
  .parse(process.env);

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
export const stripeEnv = stripeParsed.success ? stripeParsed.data : null;

const cronSchema = z.object({
  CRON_STRIPE_AUTH_KEY: z.string(),
});

const cronParsed = cronSchema.safeParse(process.env);
export const cronEnv = cronParsed.success ? cronParsed.data : null;
