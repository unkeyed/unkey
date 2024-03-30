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
  ENCRYPTION_KEYS: z.string().transform((s) =>
    z
      .array(
        z.object({
          version: z.number().int().min(1),
          key: z.string(),
        }),
      )
      .min(1)
      .parse(JSON.parse(s)),
  ),
});

export type Env = z.infer<typeof zEnv>;
