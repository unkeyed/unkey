import { z } from "zod";

export const env = () =>
  z
    .object({
      VERCEL_ENV: z
        .enum(["development", "preview", "production"])
        .optional()
        .prefault("development"),
      VERCEL_URL: z.string().optional(), // Always *.vercel.app — not the custom domain
      VERCEL_BRANCH_URL: z.string().optional(), // Only set in preview deployments
      VERCEL_PROJECT_PRODUCTION_URL: z.string().optional(), // Custom production domain (e.g. app.unkey.com)

      UNKEY_WORKSPACE_ID: z.string(),
      UNKEY_API_ID: z.string(),
      UNKEY_API_URL: z.url().optional().default("https://api.unkey.com"),
      UNKEY_JWT_SECRET: z.string().optional(),

      UPSTASH_REDIS_REST_URL: z.string().optional(),
      UPSTASH_REDIS_REST_TOKEN: z.string().optional(),

      CLERK_WEBHOOK_SECRET: z.string().optional(),
      CLERK_SECRET_KEY: z.string().optional(),
      RESEND_API_KEY: z.string().optional(),
      RESEND_AUDIENCE_ID: z.string().optional(),

      PLAIN_API_KEY: z.string().optional(),

      RATELIMIT_DEMO_ROOT_KEY: z.string().optional(),

      VAULT_URL: z.url(),
      VAULT_TOKEN: z.string(),

      CTRL_URL: z.url().optional(),
      CTRL_API_KEY: z.string().optional(),

      GITHUB_KEYS_URI: z.string().optional(),

      // This key is used for ratelimiting our trpc procedures
      // It requires the following permission:
      // `ratelimit.*.limit`
      UNKEY_ROOT_KEY: z.string().optional(),

      CLICKHOUSE_URL: z.string().optional(),
      OPENAI_API_KEY: z.string().optional(),

      AUTH_PROVIDER: z.enum(["workos", "local"]),

      WORKOS_API_KEY: z.string().optional(),
      WORKOS_CLIENT_ID: z.string().optional(),
      WORKOS_WEBHOOK_SECRET: z.string().optional(),
      NEXT_PUBLIC_WORKOS_REDIRECT_URI: z.string().optional(),
      WORKOS_COOKIE_PASSWORD: z.string().optional(),

      // Sentry configuration
      SENTRY_DISABLED: z
        .string()
        .optional()
        .transform((val) => val === "true"),
      NEXT_PUBLIC_SENTRY_DISABLED: z
        .string()
        .optional()
        .transform((val) => val === "true"),
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

export const githubAppSchema = z.object({
  GITHUB_APP_ID: z.string().transform((s) => Number.parseInt(s, 10)),
  UNKEY_GITHUB_PRIVATE_KEY_PEM: z.string().transform((s) => s.replace(/\\n/g, "\n")), // needs to be a single line, with \n
});

const githubAppParsed = githubAppSchema.safeParse(process.env);
export const githubAppEnv = () => (githubAppParsed.success ? githubAppParsed.data : null);

// GitHub App OAuth (user-to-server) credentials. These are required to verify
// that the caller who supplied an installation_id in the install callback can
// actually access that installation on GitHub before we bind it to their
// workspace. Kept separate from githubAppEnv so app-level read operations on
// already-registered installations keep working even if OAuth is unconfigured,
// while registerInstallation fails closed without it.
export const githubOAuthSchema = z.object({
  GITHUB_CLIENT_ID: z.string().min(1),
  GITHUB_CLIENT_SECRET: z.string().min(1),
});

const githubOAuthParsed = githubOAuthSchema.safeParse(process.env);
export const githubOAuthEnv = () => (githubOAuthParsed.success ? githubOAuthParsed.data : null);

const stripeSchema = z.object({
  STRIPE_SECRET_KEY: z.string(),
  // The product ids, comma separated, from lowest to highest, pro first, and then enterprise plans
  STRIPE_PRODUCT_IDS_PRO: z.string().transform((s) => s.split(",")),
  STRIPE_PRODUCT_IDS_ENTERPRISE: z.string().transform((s) => s.split(",")),
  STRIPE_WEBHOOK_SECRET: z.string(),
  // Unkey Deploy plan-fee price lookup_keys, one per plan. subscribeDeploy /
  // changeDeployPlan resolve these to the current active price and attach (or
  // swap) the plan-fee for the chosen tier. lookup_keys (not price ids) so a
  // reprice needs no env change. Optional so environments without Deploy
  // billing configured still parse; the Deploy mutations reject with a clear
  // error when unset.
  STRIPE_LOOKUP_DEPLOY_STARTER: z.string().optional(),
  STRIPE_LOOKUP_DEPLOY_PRO: z.string().optional(),
  STRIPE_LOOKUP_DEPLOY_BUSINESS: z.string().optional(),
  // Unkey Deploy metered usage price lookup_keys, shared across all plans.
  // Resolved and attached alongside the plan-fee on subscribe; usage is billed
  // from the meter events the ctrl billing worker pushes (CPU / memory /
  // egress / disk).
  STRIPE_LOOKUP_DEPLOY_METER_CPU: z.string().optional(),
  STRIPE_LOOKUP_DEPLOY_METER_MEMORY: z.string().optional(),
  STRIPE_LOOKUP_DEPLOY_METER_EGRESS: z.string().optional(),
  STRIPE_LOOKUP_DEPLOY_METER_DISK: z.string().optional(),
  // Dev/test only: create Stripe customers under a test clock so the billing
  // lifecycle can be time-traveled (advance the clock, invoices finalize for
  // real, PDFs exist). Requires a test-mode key; see
  // `unkey dev stripe clock` for advancing clocks and fetching invoices.
  STRIPE_DEV_TEST_CLOCK: z.string().optional(),
});

const stripeParsed = stripeSchema.safeParse(process.env);
export const stripeEnv = () => (stripeParsed.success ? stripeParsed.data : null);
