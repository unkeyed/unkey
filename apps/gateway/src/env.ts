import { z } from "zod";

export const zEnv = z.object({
  VERSION: z.string().default("unknown"),
  APEX_DOMAIN: z.string().default("unkey.io"),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  DATABASE_NAME: z.string().default("unkey"),
  CLOUDFLARE_API_KEY: z.string().optional(),
  CLOUDFLARE_ZONE_ID: z.string().optional(),
  ENVIRONMENT: z.enum(["development", "preview", "canary", "production"]).default("development"),
  TINYBIRD_TOKEN: z.string().optional(),
});

export type Env = z.infer<typeof zEnv>;
