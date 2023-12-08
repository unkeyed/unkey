import { z } from "zod";

export const unitTestEnv = z.object({
  DATABASE_HOST: z.string(),
  DATABASE_USERNAME: z.string(),
  DATABASE_PASSWORD: z.string(),
  DATABASE_MODE: z.enum(["planetscale", "mysql"]).optional().default("planetscale"),
});

export const integrationTestEnv = z.object({
  UNKEY_BASE_URL: z.string().url().default("http://127.0.0.1:8787"),
  /**
   * Make sure this key does not have any rate limits or usage limits
   * Otherwise your tests look flaky and you will be sad
   */
  UNKEY_ROOT_KEY: z.string(),
});
