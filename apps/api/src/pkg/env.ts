import { z } from "zod";
import type { MessageBody } from "./key_migration/message";

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
  TINYBIRD_PROXY_URL: z.string().optional(),
  TINYBIRD_PROXY_TOKEN: z.string().optional(),
  TINYBIRD_TOKEN: z.string().optional(),
  DO_RATELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"), // pretty loose check but it'll do I think
  DO_USAGELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"),
  KEY_MIGRATIONS: z.custom<Queue<MessageBody>>((q) => typeof q === "object"),
  EMIT_METRICS_LOGS: z
    .string()
    .optional()
    .default("true")
    .transform((v) => {
      return v === "true";
    }),
  VAULT_URL: z.string().url(),
  VAULT_TOKEN: z.string(),
});

export type Env = z.infer<typeof zEnv>;
