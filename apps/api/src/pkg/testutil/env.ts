import { z } from "zod";

export const unitTestEnv = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
});

export const integrationTestEnv = z.object({
  UNKEY_BASE_URL: z.string().url().default("http://127.0.0.1:8787"),
  UNKEY_ROOT_KEY: z.string(),
});
