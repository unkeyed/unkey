import type { Ai } from "@cloudflare/ai";
import type { VectorizeIndex } from "@cloudflare/workers-types";
import { z } from "zod";

export const zEnv = z.object({
  VERSION: z.string().default("unknown"),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  CLOUDFLARE_API_KEY: z.string().optional(),
  CLOUDFLARE_ZONE_ID: z.string().optional(),
  ENVIRONMENT: z.enum(["development", "preview", "canary", "production"]).default("development"),
  TINYBIRD_PROXY_URL: z.string().optional(),
  TINYBIRD_PROXY_TOKEN: z.string().optional(),
  TINYBIRD_TOKEN: z.string().optional(),
  APEX_DOMAIN: z.string().default("llm.unkey.io"),
  // VAULT_URL: z.string().url(),
  // VAULT_TOKEN: z.string(),
  EMIT_METRICS_LOGS: z
    .string()
    .optional()
    .default("true")
    .transform((v) => {
      return v === "true";
    }),

  VECTORIZE_INDEX: z.custom<VectorizeIndex>((v) => typeof v === "object"),
  AI: z.custom<Ai>((v) => typeof v === "object"),
});

export type Env = z.infer<typeof zEnv>;
