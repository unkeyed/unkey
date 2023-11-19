import { z } from "zod";

const Bindings = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  AXIOM_TOKEN: z.string().optional(),
  CLOUDFLARE_API_KEY: z.string().optional(),
  CLOUDFLARE_ZONE_ID: z.string().optional(),
  ENVIRONMENT: z.enum(["development", "preview", "production"]).default("development"),
  TINYBIRD_TOKEN: z.string().optional(),
});

export function checkEnv(env: Env["Bindings"]): void {
  Bindings.parse(env);
}

export type Env = {
  Bindings: z.infer<typeof Bindings> & {
    DO_RATELIMIT: DurableObjectNamespace;
    DO_USAGELIMIT: DurableObjectNamespace;
  };
};
