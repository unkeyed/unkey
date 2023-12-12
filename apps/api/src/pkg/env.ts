import { z } from "zod";

export const zEnv = z.object({
  VERSION: z.string().default("unknown"),
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  DATABASE_NAME: z.string().default("unkey"),
  AXIOM_TOKEN: z.string().optional(),
  CLOUDFLARE_API_KEY: z.string().optional(),
  CLOUDFLARE_ZONE_ID: z.string().optional(),
  ENVIRONMENT: z.enum(["development", "preview", "production"]).default("development"),
  TINYBIRD_TOKEN: z.string().optional(),
  DO_RATELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"), // pretty loose check but it'll do I think
  DO_USAGELIMIT: z.custom<DurableObjectNamespace>((ns) => typeof ns === "object"),
});

export type Env = z.infer<typeof zEnv>;
