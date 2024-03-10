import { z } from "zod";

export const databaseEnv = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
});

export const routeTestEnv = databaseEnv.merge(
  z.object({
    WORKER_LOCATION: z.enum(["local", "cloudflare"]).default("local"),
  }),
);

export const integrationTestEnv = databaseEnv.merge(
  z.object({
    UNKEY_BASE_URL: z.string().url().default("http://127.0.0.1:8787"),
  }),
);

export const benchmarkTestEnv = databaseEnv.merge(
  z.object({
    PLANETFALL_URL: z.string().url(),
    PLANETFALL_API_KEY: z.string(),
    UNKEY_BASE_URL: z.string().url(),
  }),
);
