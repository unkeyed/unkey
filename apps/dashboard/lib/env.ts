import { z } from "zod";

export const env = () =>
  z
    .object({
      VERCEL_ENV: z
        .enum(["development", "preview", "production"])
        .optional()
        .default("development"),
      VERCEL_URL: z.string().optional(),

      UNKEY_WORKSPACE_ID: z.string(),
      UNKEY_API_ID: z.string(),
      UNKEY_WEBHOOK_KEYS_API_ID: z.string(),

      UPSTASH_REDIS_REST_URL: z.string().optional(),
      UPSTASH_REDIS_REST_TOKEN: z.string().optional(),

      TINYBIRD_TOKEN: z.string().optional(),

      CLERK_WEBHOOK_SECRET: z.string().optional(),
      CLERK_SECRET_KEY: z.string().optional(),
      RESEND_API_KEY: z.string().optional(),
      RESEND_AUDIENCE_ID: z.string().optional(),

      PLAIN_API_KEY: z.string().optional(),

      RATELIMIT_DEMO_ROOT_KEY: z.string().optional(),

      AGENT_URL: z.string().url(),
      AGENT_TOKEN: z.string(),

      GITHUB_KEYS_URI: z.string().optional(),

      // This key is used for ratelimiting our trpc procedures
      // It requires the following permissions:
      // - `ratelimit.*.create_namespace`
      // - `ratelimit.*.limit`
      UNKEY_ROOT_KEY: z.string().optional(),

      CLICKHOUSE_URL: z.string().optional(),
    })
    .parse(process.env);

export const dbEnv = () =>
  z
    .object({
      DATABASE_HOST: z.string(),
      DATABASE_USERNAME: z.string(),
      DATABASE_PASSWORD: z.string(),
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
  STRIPE_PRODUCT_ID_KEY_VERIFICATIONS: z.string(),
  STRIPE_PRODUCT_ID_PRO_PLAN: z.string(),
  STRIPE_PRODUCT_ID_SUPPORT: z.string(),
});

const stripeParsed = stripeSchema.safeParse(process.env);
export const stripeEnv = () => (stripeParsed.success ? stripeParsed.data : null);
