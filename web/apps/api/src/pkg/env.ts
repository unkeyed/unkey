import { z } from "zod";
import type { MessageBody } from "./key_migration/message";

export const cloudflareRatelimiter = z.custom<{
  limit: (opts: { key: string }) => Promise<{ success: boolean }>;
}>((r) => !!r && typeof r.limit === "function");

export const zEnv = z.object({
  VERSION: z.string().default("unknown"),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  DATABASE_NAME: z.string().default("unkey"),
  DATABASE_HOST_READONLY: z.string().optional(),
  DATABASE_USERNAME_READONLY: z.string().optional(),
  DATABASE_PASSWORD_READONLY: z.string().optional(),
  AXIOM_TOKEN: z.string().optional(),
  CLOUDFLARE_API_KEY: z.string().optional(),
  CLOUDFLARE_ZONE_ID: z.string().optional(),
  ENVIRONMENT: z.enum(["development", "preview", "canary", "production"]).default("development"),
  DO_RATELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"), // pretty loose check but it'll do I think
  DO_USAGELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"),
  KEY_MIGRATIONS: z.custom<Queue<MessageBody>>((q) => typeof q === "object").optional(),
  EMIT_METRICS_LOGS: z
    .string()
    .optional()
    .default("true")
    .transform((v) => {
      return v === "true";
    }),
  AGENT_URL: z.string().url(),
  AGENT_TOKEN: z.string(),

  CLICKHOUSE_URL: z.string(),
  CLICKHOUSE_PROXY_URL: z.string().optional(),
  CLICKHOUSE_PROXY_TOKEN: z.string().optional(),

  UNKEY_ROOT_KEY: z.string().optional(),

  SYNC_RATELIMIT_ON_NO_DATA: z
    .string()
    .optional()
    .default("0")
    .transform((s) => {
      try {
        return Number.parseFloat(s) || 0;
      } catch {
        return 0;
      }
    }),
  RL_10_60s: cloudflareRatelimiter,
  RL_30_60s: cloudflareRatelimiter,
  RL_50_60s: cloudflareRatelimiter,
  RL_200_60s: cloudflareRatelimiter,
  RL_600_60s: cloudflareRatelimiter,
  RL_1_10s: cloudflareRatelimiter,
  RL_500_10s: cloudflareRatelimiter,
  RL_200_10s: cloudflareRatelimiter,
});

export type Env = z.infer<typeof zEnv>;
